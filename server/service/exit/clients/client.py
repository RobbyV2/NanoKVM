#!/usr/bin/env python3
# nexit/1 python client
#
# NanoKVM exit client: holds one WebSocket open to the NanoKVM and originates
# every TCP connection and UDP datagram the consumer behind it asks for.
# Served fully templated from GET /exit/<slot>/client.py; takes no arguments.
# Python 3.8+, standard library only.
#
# Protocol: docs/superpowers/specs/2026-09-15-exit-tunnel-design.md, section
# "The nexit/1 protocol (Mode A)". Frames are type:u8 stream:u32be payload,
# exactly one per WebSocket binary message, at most 16 KiB of payload.
# HELLO's hostname and os strings carry a one-byte length prefix.

import base64
import errno
import hashlib
import ipaddress
import os
import platform
import selectors
import signal
import socket
import ssl
import struct
import sys
import time
from collections import deque

SCHEME = "__SCHEME__"
HOST = "__HOST__"
SLOT = "__SLOT__"
TOKEN = "__TOKEN__"
VERIFY = "__VERIFY__" == "1"
ALLOW_PRIVATE = "__ALLOW_PRIVATE__" == "1"

# Frame types.
T_HELLO, T_WELCOME = 0x01, 0x02
T_OPEN, T_OPENED, T_OPEN_FAIL = 0x10, 0x11, 0x12
T_DATA, T_EOF, T_RST, T_WINDOW = 0x20, 0x21, 0x22, 0x30

# SOCKS5 REP codes reused as reasons.
R_GENERAL, R_NOT_ALLOWED, R_NET_UNREACH, R_HOST_UNREACH = 1, 2, 3, 4
R_REFUSED, R_TIMEOUT, R_CMD_UNSUPP, R_ATYP_UNSUPP = 5, 6, 7, 8

MAX_PAYLOAD = 16 * 1024
WS_READ_LIMIT = MAX_PAYLOAD + 64
CONNECT_BUDGET = 6.0
PING_EVERY = 30.0
DEAD_AFTER = 75.0
UDP_QUEUE = 32
WS_GUID = b"258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
EV_READ, EV_WRITE = selectors.EVENT_READ, selectors.EVENT_WRITE

ALWAYS4 = [ipaddress.ip_network(p) for p in (
    "0.0.0.0/8", "127.0.0.0/8", "169.254.0.0/16", "198.18.0.0/15", "224.0.0.0/3")]
PRIVATE4 = [ipaddress.ip_network(p) for p in (
    "10.0.0.0/8", "100.64.0.0/10", "172.16.0.0/12", "192.168.0.0/16")]
ALWAYS6 = [ipaddress.ip_network(p) for p in ("::/128", "::1/128", "fe80::/10", "ff00::/8")]
PRIVATE6 = [ipaddress.ip_network(p) for p in ("fc00::/7",)]

ERRNO_REASON = {}
for _name, _rep in (("ECONNREFUSED", R_REFUSED), ("EHOSTUNREACH", R_HOST_UNREACH),
                    ("ENETUNREACH", R_NET_UNREACH), ("ETIMEDOUT", R_TIMEOUT),
                    ("EADDRNOTAVAIL", R_ATYP_UNSUPP), ("EAFNOSUPPORT", R_ATYP_UNSUPP),
                    ("WSAECONNREFUSED", R_REFUSED), ("WSAEHOSTUNREACH", R_HOST_UNREACH),
                    ("WSAENETUNREACH", R_NET_UNREACH), ("WSAETIMEDOUT", R_TIMEOUT),
                    ("WSAEADDRNOTAVAIL", R_ATYP_UNSUPP), ("WSAEAFNOSUPPORT", R_ATYP_UNSUPP)):
    if hasattr(errno, _name):
        ERRNO_REASON[getattr(errno, _name)] = _rep
IN_PROGRESS = {0}
for _name in ("EINPROGRESS", "EWOULDBLOCK", "EAGAIN", "EALREADY", "WSAEWOULDBLOCK"):
    if hasattr(errno, _name):
        IN_PROGRESS.add(getattr(errno, _name))


def log(msg):
    sys.stderr.write(time.strftime("%H:%M:%S ") + "nexit: " + msg + "\n")
    sys.stderr.flush()


def policy_allows(addr):
    """Destination policy D21, mirrored from server/service/exit/policy.go."""
    if isinstance(addr, ipaddress.IPv6Address) and addr.ipv4_mapped is not None:
        addr = addr.ipv4_mapped
    if isinstance(addr, ipaddress.IPv4Address):
        always, private = ALWAYS4, PRIVATE4
    else:
        always, private = ALWAYS6, PRIVATE6
    for net in always:
        if addr in net:
            return False
    if ALLOW_PRIVATE:
        return True
    for net in private:
        if addr in net:
            return False
    return True


def parse_host(hostport, default_port):
    if hostport.startswith("["):
        end = hostport.find("]")
        rest = hostport[end + 1:]
        return hostport[1:end], (int(rest[1:]) if rest.startswith(":") else default_port)
    if hostport.count(":") == 1:
        host, port = hostport.split(":")
        return host, int(port)
    return hostport, default_port


def parse_addr(payload, off):
    """Parse atyp, addr, port at payload[off:]; returns (ip, port, newoff)."""
    atyp = payload[off]
    if atyp == 1:
        ip = ipaddress.IPv4Address(bytes(payload[off + 1:off + 5]))
        off += 5
    elif atyp == 4:
        ip = ipaddress.IPv6Address(bytes(payload[off + 1:off + 17]))
        off += 17
    else:
        raise ValueError("atyp %d" % atyp)
    port = struct.unpack("!H", payload[off:off + 2])[0]
    return ip, port, off + 2


def pack_addr(ip, port):
    atyp = b"\x01" if isinstance(ip, ipaddress.IPv4Address) else b"\x04"
    return atyp + ip.packed + struct.pack("!H", port)


def sockname(sock):
    name = sock.getsockname()
    ip = ipaddress.ip_address(name[0].split("%")[0])
    if isinstance(ip, ipaddress.IPv6Address) and ip.ipv4_mapped is not None:
        ip = ip.ipv4_mapped
    return ip, name[1]


def has_default_route():
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        try:
            s.connect(("1.1.1.1", 53))  # no packet is sent; only the route lookup happens
            return True
        finally:
            s.close()
    except OSError:
        return False


def os_name():
    n = platform.system().lower()
    return {"darwin": "darwin", "linux": "linux", "windows": "windows"}.get(n, n or "unknown")


def ws_mask(payload, key):
    n = len(payload)
    if n == 0:
        return b""
    full = (key * ((n + 3) // 4))[:n]
    return (int.from_bytes(payload, "big") ^ int.from_bytes(full, "big")).to_bytes(n, "big")


def ws_frame(opcode, payload):
    n = len(payload)
    head = bytearray([0x80 | opcode])
    if n < 126:
        head.append(0x80 | n)
    elif n < 65536:
        head.append(0x80 | 126)
        head += struct.pack("!H", n)
    else:
        head.append(0x80 | 127)
        head += struct.pack("!Q", n)
    key = os.urandom(4)
    return bytes(head) + key + ws_mask(payload, key)


def nexit(ftype, sid, payload=b""):
    return struct.pack("!BI", ftype, sid) + payload


class SessionEnd(Exception):
    pass


class Stream(object):
    __slots__ = ("sid", "proto", "sock", "state", "deadline", "tx_credit", "rx_unacked",
                 "wbuf", "peer_eof", "shut", "read_eof", "udp_queued", "events")

    def __init__(self, sid, proto):
        self.sid = sid
        self.proto = proto          # 1 tcp, 2 udp
        self.sock = None
        self.state = "connecting"   # then "open"
        self.deadline = 0.0
        self.tx_credit = 0          # bytes we may still send toward the kvm
        self.rx_unacked = 0         # bytes written to the socket since the last WINDOW
        self.wbuf = bytearray()     # kvm -> socket, not yet written
        self.peer_eof = False       # kvm sent EOF
        self.shut = False           # we shut down our write side
        self.read_eof = False       # socket hit EOF, we sent EOF
        self.udp_queued = 0         # UDP DATA messages of this stream waiting in `out`
        self.events = 0             # selector interest currently registered (0 = none)


class Exit(object):
    def __init__(self, ws, is_tls, leftover):
        self.ws = ws
        self.is_tls = is_tls
        self.sel = selectors.DefaultSelector()
        self.streams = {}
        self.inbuf = bytearray(leftover)
        self.out = deque()          # (stream id or 0, WebSocket message bytes)
        self.out_off = 0            # bytes of out[0] already written
        self.frag_op = None
        self.frag_buf = bytearray()
        self.stream_window = 0
        self.conn_window = 0
        self.max_streams = 0
        self.conn_credit = 0        # bytes we may still send on TCP streams toward the kvm
        self.conn_unacked = 0       # bytes consumed since the last WINDOW on stream 0
        self.welcomed = False
        self.last_rx = time.monotonic()
        self.last_ping = time.monotonic()
        self.ws_events = EV_READ
        self.sel.register(ws, EV_READ, self)

    # --- outbound ------------------------------------------------------------

    def send_ws(self, opcode, payload, sid=0):
        self.out.append((sid, ws_frame(opcode, payload)))
        self.want_ws_write()

    def send(self, ftype, sid, payload=b""):
        self.send_ws(0x2, nexit(ftype, sid, payload))

    def send_udp_data(self, st, payload):
        if st.udp_queued >= UDP_QUEUE:
            # drop the oldest unsent datagram of this stream, never a half-written head
            for i in range(1 if self.out_off else 0, len(self.out)):
                if self.out[i][0] == st.sid:
                    del self.out[i]
                    st.udp_queued -= 1
                    break
        st.udp_queued += 1
        self.send_ws(0x2, nexit(T_DATA, st.sid, payload), st.sid)

    def want_ws_write(self):
        ev = EV_READ | (EV_WRITE if self.out else 0)
        if ev != self.ws_events:
            self.ws_events = ev
            self.sel.modify(self.ws, ev, self)

    def flush_ws(self):
        while self.out:
            sid, msg = self.out[0]
            try:
                n = self.ws.send(msg[self.out_off:])
            except (ssl.SSLWantWriteError, ssl.SSLWantReadError, BlockingIOError):
                break
            except InterruptedError:
                continue
            except OSError as e:
                raise SessionEnd("websocket write: %s" % e)
            self.out_off += n
            if self.out_off >= len(msg):
                self.out.popleft()
                self.out_off = 0
                if sid and sid in self.streams:
                    self.streams[sid].udp_queued -= 1
        self.want_ws_write()

    # --- inbound WebSocket ---------------------------------------------------

    def read_ws(self):
        while True:
            try:
                chunk = self.ws.recv(65536)
            except (ssl.SSLWantReadError, ssl.SSLWantWriteError, BlockingIOError):
                break
            except InterruptedError:
                continue
            except OSError as e:
                raise SessionEnd("websocket read: %s" % e)
            if not chunk:
                raise SessionEnd("websocket closed by peer")
            self.inbuf += chunk
            if len(chunk) < 65536 and not (self.is_tls and self.ws.pending()):
                break
        self.parse_ws()

    def parse_ws(self):
        buf = self.inbuf
        while True:
            if len(buf) < 2:
                return
            b0, b1 = buf[0], buf[1]
            fin, opcode = b0 & 0x80, b0 & 0x0F
            if b0 & 0x70:
                raise SessionEnd("websocket: reserved bits set")
            if b1 & 0x80:
                raise SessionEnd("websocket: masked frame from server")
            ln, off = b1 & 0x7F, 2
            if ln == 126:
                if len(buf) < 4:
                    return
                ln, off = struct.unpack("!H", buf[2:4])[0], 4
            elif ln == 127:
                if len(buf) < 10:
                    return
                ln, off = struct.unpack("!Q", buf[2:10])[0], 10
            if ln + len(self.frag_buf) > WS_READ_LIMIT:
                raise SessionEnd("websocket: message over read limit (%d)" % ln)
            if len(buf) < off + ln:
                return
            payload = bytes(buf[off:off + ln])
            del buf[:off + ln]
            self.last_rx = time.monotonic()
            if opcode == 0x8:
                code = struct.unpack("!H", payload[:2])[0] if len(payload) >= 2 else 1005
                raise SessionEnd("websocket closed by kvm (%d %s)" % (
                    code, payload[2:].decode("utf-8", "replace")))
            if opcode == 0x9:
                self.send_ws(0xA, payload)
                continue
            if opcode == 0xA:
                continue
            if opcode == 0x1:
                raise SessionEnd("websocket: text message is a protocol error")
            if opcode == 0x0:
                if self.frag_op is None:
                    raise SessionEnd("websocket: continuation without start")
                self.frag_buf += payload
                if not fin:
                    continue
                opcode, payload = self.frag_op, bytes(self.frag_buf)
                self.frag_op, self.frag_buf = None, bytearray()
            elif opcode == 0x2:
                if self.frag_op is not None:
                    raise SessionEnd("websocket: interleaved data frames")
                if not fin:
                    self.frag_op, self.frag_buf = opcode, bytearray(payload)
                    continue
            else:
                raise SessionEnd("websocket: unknown opcode %d" % opcode)
            self.on_message(payload)

    # --- nexit frames --------------------------------------------------------

    def on_message(self, msg):
        if len(msg) < 5:
            raise SessionEnd("short frame")
        ftype, sid = struct.unpack("!BI", msg[:5])
        payload = msg[5:]
        if not self.welcomed:
            if ftype != T_WELCOME or len(payload) < 11:
                raise SessionEnd("expected WELCOME, got 0x%02x" % ftype)
            ver, sw, cw, ms = struct.unpack("!BIIH", payload[:11])
            if ver != 1:
                raise SessionEnd("unsupported version %d" % ver)
            self.stream_window, self.conn_window, self.max_streams = sw, cw, ms
            self.conn_credit = cw
            self.welcomed = True
            log("connected: streamWindow=%d connWindow=%d maxStreams=%d" % (sw, cw, ms))
            return
        if ftype == T_OPEN:
            self.on_open(sid, payload)
            return
        if ftype == T_WINDOW:
            if len(payload) < 4:
                raise SessionEnd("short WINDOW")
            inc = struct.unpack("!I", payload[:4])[0]
            if sid == 0:
                self.conn_credit += inc
                for st in list(self.streams.values()):
                    self.refresh(st)
            else:
                st = self.streams.get(sid)
                if st is not None:
                    st.tx_credit += inc
                    self.refresh(st)
            return
        st = self.streams.get(sid)
        if st is None:
            return  # released or never existed: ignored by design
        if ftype == T_DATA:
            if st.state != "open":
                raise SessionEnd("DATA on stream %d before OPENED" % sid)
            if st.proto == 2:
                self.udp_out(st, payload)
            elif st.peer_eof:
                self.rst(st, R_GENERAL)
            else:
                st.wbuf += payload
                self.drain(st)
        elif ftype == T_EOF:
            if st.proto == 2 or st.state != "open" or st.peer_eof:
                self.rst(st, R_GENERAL)
            else:
                st.peer_eof = True
                self.drain(st)
        elif ftype == T_RST:
            self.release(st)
        else:
            raise SessionEnd("unexpected frame 0x%02x on stream %d" % (ftype, sid))

    def on_open(self, sid, payload):
        if sid in self.streams or sid == 0 or sid % 2 == 0:
            raise SessionEnd("OPEN on live or invalid stream id %d" % sid)
        if len(payload) < 1:
            raise SessionEnd("short OPEN")
        proto = payload[0]
        st = Stream(sid, proto)
        st.tx_credit = self.stream_window
        self.streams[sid] = st
        if proto == 1:
            try:
                ip, port, _ = parse_addr(payload, 1)
            except (ValueError, IndexError, struct.error):
                self.open_fail(st, R_ATYP_UNSUPP)
                return
            if not policy_allows(ip):
                self.open_fail(st, R_NOT_ALLOWED)
                return
            fam = socket.AF_INET if isinstance(ip, ipaddress.IPv4Address) else socket.AF_INET6
            try:
                s = socket.socket(fam, socket.SOCK_STREAM)
            except OSError as e:
                self.open_fail(st, ERRNO_REASON.get(e.errno, R_ATYP_UNSUPP))
                return
            s.setblocking(False)
            try:
                s.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
            except OSError:
                pass
            rc = s.connect_ex((str(ip), port))
            if rc not in IN_PROGRESS:
                s.close()
                self.open_fail(st, ERRNO_REASON.get(rc, R_GENERAL))
                return
            st.sock = s
            st.deadline = time.monotonic() + CONNECT_BUDGET
            if rc == 0:
                self.connected(st)
            else:
                self.set_events(st, EV_WRITE)
        elif proto == 2:
            try:
                s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
                s.bind(("0.0.0.0", 0))
            except OSError as e:
                self.open_fail(st, ERRNO_REASON.get(e.errno, R_GENERAL))
                return
            s.setblocking(False)
            st.sock = s
            st.state = "open"
            self.set_events(st, EV_READ)
            self.send(T_OPENED, sid)
        else:
            self.open_fail(st, R_CMD_UNSUPP)

    def open_fail(self, st, reason):
        self.send(T_OPEN_FAIL, st.sid, bytes([reason]))
        self.release(st)

    def connected(self, st):
        err = st.sock.getsockopt(socket.SOL_SOCKET, socket.SO_ERROR)
        if err:
            self.open_fail(st, ERRNO_REASON.get(err, R_GENERAL))
            return
        st.state = "open"
        ip, port = sockname(st.sock)
        self.send(T_OPENED, st.sid, pack_addr(ip, port))
        self.refresh(st)

    def rst(self, st, reason):
        self.send(T_RST, st.sid, bytes([reason]))
        self.release(st)

    def release(self, st):
        self.streams.pop(st.sid, None)
        if st.sock is not None:
            self.set_events(st, 0)
            try:
                st.sock.close()
            except OSError:
                pass
        if st.proto == 1 and st.wbuf:
            # never reached the socket: hand the connection window back anyway
            self.consumed(st, len(st.wbuf), stream_too=False)
            st.wbuf = bytearray()
        if st.udp_queued:
            keep = [self.out[0]] if self.out_off else []
            keep += [item for item in list(self.out)[len(keep):] if item[0] != st.sid]
            self.out = deque(keep)
            st.udp_queued = 0
            self.want_ws_write()

    def set_events(self, st, ev):
        if ev == st.events:
            return
        if st.events == 0:
            self.sel.register(st.sock, ev, st)
        elif ev == 0:
            try:
                self.sel.unregister(st.sock)
            except (KeyError, ValueError):
                pass
        else:
            self.sel.modify(st.sock, ev, st)
        st.events = ev

    def refresh(self, st):
        """Recompute selector interest for an open stream from credit and buffers."""
        if st.sock is None or st.state != "open" or st.sid not in self.streams:
            return
        if st.proto == 2:
            self.set_events(st, EV_READ)
            return
        ev = 0
        if not st.read_eof and st.tx_credit > 0 and self.conn_credit > 0:
            ev |= EV_READ
        if st.wbuf:
            ev |= EV_WRITE
        self.set_events(st, ev)

    def consumed(self, st, n, stream_too=True):
        self.conn_unacked += n
        if self.conn_unacked >= self.conn_window // 2:
            self.send(T_WINDOW, 0, struct.pack("!I", self.conn_unacked))
            self.conn_unacked = 0
        if stream_too:
            st.rx_unacked += n
            if st.rx_unacked >= self.stream_window // 2:
                self.send(T_WINDOW, st.sid, struct.pack("!I", st.rx_unacked))
                st.rx_unacked = 0

    def drain(self, st):
        """Write pending kvm bytes into the TCP socket; shut down after EOF once empty."""
        while st.wbuf:
            try:
                n = st.sock.send(st.wbuf[:65536])
            except (BlockingIOError, InterruptedError):
                break
            except OSError as e:
                self.rst(st, ERRNO_REASON.get(e.errno, R_GENERAL))
                return
            del st.wbuf[:n]
            self.consumed(st, n)
        if not st.wbuf and st.peer_eof and not st.shut:
            st.shut = True
            try:
                st.sock.shutdown(socket.SHUT_WR)
            except OSError:
                pass
            if st.read_eof:
                self.release(st)
                return
        self.refresh(st)

    def read_stream(self, st):
        budget = min(MAX_PAYLOAD, st.tx_credit, self.conn_credit)
        if budget <= 0:
            self.refresh(st)
            return
        try:
            data = st.sock.recv(budget)
        except (BlockingIOError, InterruptedError):
            return
        except OSError as e:
            self.rst(st, ERRNO_REASON.get(e.errno, R_GENERAL))
            return
        if not data:
            st.read_eof = True
            self.send(T_EOF, st.sid)
            if st.shut:
                self.release(st)
            else:
                self.refresh(st)
            return
        st.tx_credit -= len(data)
        self.conn_credit -= len(data)
        self.send(T_DATA, st.sid, data)
        if self.conn_credit <= 0:
            for other in list(self.streams.values()):
                self.refresh(other)
        else:
            self.refresh(st)

    def udp_out(self, st, payload):
        try:
            ip, port, off = parse_addr(payload, 0)
        except (ValueError, IndexError, struct.error):
            return
        if not isinstance(ip, ipaddress.IPv4Address) or not policy_allows(ip):
            return
        try:
            st.sock.sendto(payload[off:], (str(ip), port))
        except OSError:
            pass

    def udp_in(self, st):
        try:
            data, src = st.sock.recvfrom(65535)
        except (BlockingIOError, InterruptedError):
            return
        except OSError as e:
            self.rst(st, ERRNO_REASON.get(e.errno, R_GENERAL))
            return
        head = pack_addr(ipaddress.ip_address(src[0].split("%")[0]), src[1])
        if len(head) + len(data) > MAX_PAYLOAD:
            return
        self.send_udp_data(st, head + data)

    # --- the loop ------------------------------------------------------------

    def run(self):
        self.send(T_HELLO, 0, self.hello())
        self.parse_ws()
        while True:
            now = time.monotonic()
            if now - self.last_rx > DEAD_AFTER:
                raise SessionEnd("no frame from kvm for %ds" % DEAD_AFTER)
            if now - self.last_ping >= PING_EVERY:
                self.last_ping = now
                self.send_ws(0x9, b"nexit")
            timeout = min(1.0, PING_EVERY - (now - self.last_ping), DEAD_AFTER - (now - self.last_rx))
            for st in list(self.streams.values()):
                if st.state == "connecting" and st.proto == 1:
                    if st.deadline <= now:
                        self.open_fail(st, R_TIMEOUT)
                    else:
                        timeout = min(timeout, st.deadline - now)
            if self.out:
                self.flush_ws()
            for key, ev in self.sel.select(max(timeout, 0.01)):
                obj = key.data
                if obj is self:
                    if ev & EV_WRITE:
                        self.flush_ws()
                    if ev & EV_READ:
                        self.read_ws()
                    continue
                st = obj
                if st.sid not in self.streams:
                    continue
                if st.state == "connecting":
                    self.connected(st)
                elif st.proto == 2:
                    self.udp_in(st)
                else:
                    if ev & EV_WRITE and st.wbuf:
                        self.drain(st)
                    if st.sid in self.streams and ev & EV_READ:
                        self.read_stream(st)

    def hello(self):
        host = socket.gethostname().encode("utf-8", "replace")[:255]
        osn = os_name().encode("utf-8")[:255]
        flags = 1 if has_default_route() else 0
        return bytes([1, flags, len(host)]) + host + bytes([len(osn)]) + osn

    def close(self, reason):
        for st in list(self.streams.values()):
            self.release(st)
        try:
            self.ws.settimeout(2.0)
            self.ws.sendall(ws_frame(0x8, struct.pack("!H", 1000) + reason.encode("utf-8")[:100]))
        except (OSError, ValueError):
            pass
        try:
            self.ws.close()
        except OSError:
            pass
        self.sel.close()


def connect_ws():
    """Dial, TLS, HTTP upgrade. Returns (socket, is_tls, leftover)."""
    scheme = {"http": "ws", "https": "wss"}.get(SCHEME, SCHEME)
    if scheme not in ("ws", "wss"):
        raise RuntimeError("bad scheme %r" % SCHEME)
    tls = scheme == "wss"
    host, port = parse_host(HOST, 443 if tls else 80)
    sock = socket.create_connection((host, port), timeout=10)
    try:
        sock.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
        if tls:
            ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
            sni = None
            try:
                ipaddress.ip_address(host)
            except ValueError:
                sni = host
            if VERIFY:
                # A CA-signed certificate is installed, so check it the ordinary way.
                ctx.load_default_certs(ssl.Purpose.SERVER_AUTH)
                ctx.check_hostname = sni is not None
            else:
                # The unit's own certificate is self-signed and no CA store can
                # vouch for it. Trust the transport, as wstunnel does against the
                # same host, and let the token authenticate the session.
                ctx.check_hostname = False
                ctx.verify_mode = ssl.CERT_NONE
            sock = ctx.wrap_socket(sock, server_hostname=sni)
        key = base64.b64encode(os.urandom(16)).decode()
        req = ("GET /exit/%s/native HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"
               "Sec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\nAuthorization: Bearer %s\r\n"
               "User-Agent: nexit-python/1\r\n\r\n") % (SLOT, HOST, key, TOKEN)
        sock.sendall(req.encode())
        buf = b""
        while b"\r\n\r\n" not in buf:
            chunk = sock.recv(4096)
            if not chunk:
                raise RuntimeError("connection closed during handshake")
            buf += chunk
            if len(buf) > 65536:
                raise RuntimeError("oversized handshake response")
        head, rest = buf.split(b"\r\n\r\n", 1)
        lines = head.decode("iso-8859-1").split("\r\n")
        status = lines[0].split(" ", 2)
        if len(status) < 2 or status[1] != "101":
            if len(status) > 1 and status[1] == "404":
                raise RuntimeError("rejected (404): wrong token or slot, or this source is rate limited")
            raise RuntimeError("upgrade refused: %s" % lines[0])
        headers = {}
        for line in lines[1:]:
            if ":" in line:
                k, v = line.split(":", 1)
                headers[k.strip().lower()] = v.strip()
        expect = base64.b64encode(hashlib.sha1(key.encode() + WS_GUID).digest()).decode()
        if headers.get("sec-websocket-accept") != expect:
            raise RuntimeError("bad Sec-WebSocket-Accept")
        if headers.get("upgrade", "").lower() != "websocket":
            raise RuntimeError("no Upgrade: websocket in response")
        sock.setblocking(False)
        return sock, tls, rest
    except BaseException:
        sock.close()
        raise


def main():
    if HOST.startswith("__") or TOKEN.startswith("__"):
        log("this script must be fetched from the NanoKVM, which fills in its parameters")
        return 2

    def on_signal(signum, frame):
        raise KeyboardInterrupt()

    signal.signal(signal.SIGINT, on_signal)
    if hasattr(signal, "SIGTERM"):
        signal.signal(signal.SIGTERM, on_signal)
    if hasattr(signal, "SIGPIPE"):
        signal.signal(signal.SIGPIPE, signal.SIG_IGN)
    scheme = {"http": "ws", "https": "wss"}.get(SCHEME, SCHEME)
    log("exit client for %s://%s/exit/%s (allowPrivate=%s, verifyTLS=%s)" % (
        scheme, HOST, SLOT, ALLOW_PRIVATE, VERIFY))
    backoff = 1
    while True:
        ex = None
        try:
            ws, tls, rest = connect_ws()
            ex = Exit(ws, tls, rest)
            try:
                ex.run()
            except SessionEnd as e:
                if ex.welcomed:
                    backoff = 1
                log("session ended: %s" % e)
                ex.close(str(e)[:100])
        except KeyboardInterrupt:
            if ex is not None:
                ex.close("client exiting")
            log("stopping")
            return 0
        except (OSError, RuntimeError, ssl.SSLError) as e:
            log("connect failed: %s" % e)
        log("reconnecting in %ds" % backoff)
        try:
            time.sleep(backoff)
        except KeyboardInterrupt:
            log("stopping")
            return 0
        backoff = min(backoff * 2, 30)


if __name__ == "__main__":
    sys.exit(main())

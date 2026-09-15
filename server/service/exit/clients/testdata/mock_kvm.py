#!/usr/bin/env python3
"""mock_kvm.py: the NanoKVM side of nexit/1 for exercising the exit clients on a host.

It is a WebSocket server implementing the kvm half of the protocol in
docs/superpowers/specs/2026-09-15-exit-tunnel-design.md plus a SOCKS5 front
door on loopback, so `curl --socks5-hostname 127.0.0.1:<port>` and a UDP
ASSOCIATE client can be driven through client.py / client.pl / client.ps1
without the Go server. Unlike hev, tools like curl send domain names; the mock
resolves them itself and only ever puts atyp 1/4 on the wire, like the real
front door which answers domain CONNECTs with 0x08.

    python3 mock_kvm.py [--listen 127.0.0.1:0] [--socks 127.0.0.1:0] [--token t]
                        [--slot 0] [--tls] [--allow-private] [-v]

Test aid only; not part of the shipped server.
"""
import argparse
import base64
import hashlib
import ipaddress
import os
import queue
import socket
import ssl
import struct
import subprocess
import sys
import tempfile
import threading
import time

T_HELLO, T_WELCOME = 0x01, 0x02
T_OPEN, T_OPENED, T_OPEN_FAIL = 0x10, 0x11, 0x12
T_DATA, T_EOF, T_RST, T_WINDOW = 0x20, 0x21, 0x22, 0x30
STREAM_WINDOW, CONN_WINDOW, MAX_STREAMS, MAX_PAYLOAD = 128 << 10, 4 << 20, 256, 16 << 10
WS_READ_LIMIT = MAX_PAYLOAD + 64
WS_GUID = b"258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
NAMES = {0x01: "HELLO", 0x02: "WELCOME", 0x10: "OPEN", 0x11: "OPENED", 0x12: "OPEN_FAIL",
         0x20: "DATA", 0x21: "EOF", 0x22: "RST", 0x30: "WINDOW"}

ALWAYS4 = [ipaddress.ip_network(p) for p in ("0.0.0.0/8", "127.0.0.0/8", "169.254.0.0/16", "198.18.0.0/15", "224.0.0.0/3")]
PRIVATE4 = [ipaddress.ip_network(p) for p in ("10.0.0.0/8", "100.64.0.0/10", "172.16.0.0/12", "192.168.0.0/16")]
ALWAYS6 = [ipaddress.ip_network(p) for p in ("::/128", "::1/128", "fe80::/10", "ff00::/8")]
PRIVATE6 = [ipaddress.ip_network(p) for p in ("fc00::/7",)]

ARGS = None
LOG_LOCK = threading.Lock()


def log(msg, verbose=False):
    if verbose and not ARGS.verbose:
        return
    with LOG_LOCK:
        sys.stderr.write(time.strftime("%H:%M:%S ") + "mock_kvm: " + msg + "\n")
        sys.stderr.flush()


def policy_allows(ip):
    if isinstance(ip, ipaddress.IPv6Address) and ip.ipv4_mapped is not None:
        ip = ip.ipv4_mapped
    always, private = (ALWAYS4, PRIVATE4) if isinstance(ip, ipaddress.IPv4Address) else (ALWAYS6, PRIVATE6)
    if any(ip in n for n in always):
        return False
    return ARGS.allow_private or not any(ip in n for n in private)


def pack_addr(ip, port):
    return (b"\x01" if isinstance(ip, ipaddress.IPv4Address) else b"\x04") + ip.packed + struct.pack("!H", port)


def parse_addr(buf, off):
    atyp = buf[off]
    if atyp == 1:
        ip, off = ipaddress.IPv4Address(bytes(buf[off + 1:off + 5])), off + 5
    elif atyp == 4:
        ip, off = ipaddress.IPv6Address(bytes(buf[off + 1:off + 17])), off + 17
    else:
        raise ValueError("atyp %d" % atyp)
    return ip, struct.unpack("!H", buf[off:off + 2])[0], off + 2


def resolve(name, port):
    for fam in (socket.AF_INET, socket.AF_INET6):
        try:
            info = socket.getaddrinfo(name, port, fam, socket.SOCK_STREAM)
        except socket.gaierror:
            continue
        if info:
            return ipaddress.ip_address(info[0][4][0].split("%")[0])
    return None


class SessionGone(Exception):
    pass


class KStream(object):
    def __init__(self, sid, proto):
        self.sid, self.proto = sid, proto
        self.opened = threading.Event()
        self.fail = None            # reason byte on OPEN_FAIL
        self.bound = None
        self.rx = queue.Queue()     # bytes DATA, b"" EOF, None RST
        self.tx_credit = STREAM_WINDOW
        self.rx_unacked = 0
        self.released = False


class Session(object):
    """One attached exit."""

    def __init__(self, sock, peer):
        self.sock, self.peer = sock, peer
        self.wlock = threading.Lock()
        self.cond = threading.Condition()
        self.conn_credit = CONN_WINDOW
        self.conn_unacked = 0
        self.streams = {}
        self.next_sid = 1
        self.closed = threading.Event()
        self.pong_at = time.monotonic()
        self.hostname = self.os = ""

    # websocket layer ----------------------------------------------------------
    def send_raw(self, opcode, payload):
        n = len(payload)
        head = bytearray([0x80 | opcode])
        if n < 126:
            head.append(n)
        elif n < 65536:
            head.append(126)
            head += struct.pack("!H", n)
        else:
            raise ValueError("kvm never emits the 8-byte length form")
        with self.wlock:
            try:
                self.sock.sendall(bytes(head) + payload)
            except OSError as e:
                self.close("write: %s" % e)
                raise SessionGone()

    def send(self, ftype, sid, payload=b""):
        log("-> %s stream=%d len=%d" % (NAMES.get(ftype, hex(ftype)), sid, len(payload)), verbose=True)
        self.send_raw(0x2, struct.pack("!BI", ftype, sid) + payload)

    def read_exact(self, n):
        buf = b""
        while len(buf) < n:
            chunk = self.sock.recv(n - len(buf))
            if not chunk:
                raise SessionGone()
            buf += chunk
        return buf

    def read_message(self):
        """One complete data message (fragments reassembled); handles control frames inline."""
        frag_op, frag = None, bytearray()
        while True:
            b0, b1 = self.read_exact(2)
            fin, opcode, masked, ln = b0 & 0x80, b0 & 0x0F, b1 & 0x80, b1 & 0x7F
            if ln == 126:
                ln = struct.unpack("!H", self.read_exact(2))[0]
            elif ln == 127:
                ln = struct.unpack("!Q", self.read_exact(8))[0]
            if ln + len(frag) > WS_READ_LIMIT:
                self.close("message over read limit", code=1009)
                raise SessionGone()
            if not masked:
                self.close("client frame not masked", code=1002)
                raise SessionGone()
            key = self.read_exact(4)
            payload = self.read_exact(ln)
            if ln:
                full = (key * ((ln + 3) // 4))[:ln]
                payload = (int.from_bytes(payload, "big") ^ int.from_bytes(full, "big")).to_bytes(ln, "big")
            if opcode == 0x8:
                code = struct.unpack("!H", payload[:2])[0] if len(payload) >= 2 else 1005
                log("exit closed the websocket (%d %s)" % (code, payload[2:].decode("utf-8", "replace")))
                self.close("peer close", code=1000)
                raise SessionGone()
            if opcode == 0x9:
                self.send_raw(0xA, payload)
                continue
            if opcode == 0xA:
                self.pong_at = time.monotonic()
                continue
            if opcode == 0x1:
                self.close("text message", code=1003)
                raise SessionGone()
            if opcode == 0x0:
                if frag_op is None:
                    self.close("continuation without start", code=1002)
                    raise SessionGone()
                frag += payload
                if fin:
                    return bytes(frag)
                continue
            if opcode == 0x2:
                if fin:
                    return payload
                frag_op, frag = opcode, bytearray(payload)
                continue
            self.close("unknown opcode %d" % opcode, code=1002)
            raise SessionGone()

    def close(self, reason, code=1000):
        if self.closed.is_set():
            return
        self.closed.set()
        log("session from %s closed: %s" % (self.peer, reason))
        try:
            with self.wlock:
                self.sock.settimeout(1.0)
                self.sock.sendall(b"\x88" + bytes([2 + len(reason[:100])]) + struct.pack("!H", code) + reason[:100].encode())
        except OSError:
            pass
        try:
            self.sock.close()
        except OSError:
            pass
        with self.cond:
            for st in list(self.streams.values()):
                st.rx.put(None)
                st.fail = st.fail or 0x03
                st.opened.set()
            self.cond.notify_all()

    # nexit layer --------------------------------------------------------------
    def handshake(self):
        self.sock.settimeout(5.0)
        msg = self.read_message()
        if len(msg) < 7 or msg[0] != T_HELLO or struct.unpack("!I", msg[1:5])[0] != 0:
            self.close("first frame is not HELLO", code=1002)
            raise SessionGone()
        p = msg[5:]
        ver, flags = p[0], p[1]
        hl = p[2]
        self.hostname = p[3:3 + hl].decode("utf-8", "replace")
        ol = p[3 + hl]
        self.os = p[4 + hl:4 + hl + ol].decode("utf-8", "replace")
        log("HELLO version=%d flags=%#x hostname=%r os=%r" % (ver, flags, self.hostname, self.os))
        if ver != 1:
            self.close("unsupported version", code=1002)
            raise SessionGone()
        self.sock.settimeout(None)
        self.send(T_WELCOME, 0, struct.pack("!BIIH", 1, STREAM_WINDOW, CONN_WINDOW, MAX_STREAMS))

    def reader(self):
        try:
            while not self.closed.is_set():
                msg = self.read_message()
                if len(msg) < 5:
                    self.close("short frame", code=1002)
                    return
                ftype, sid = struct.unpack("!BI", msg[:5])
                payload = msg[5:]
                log("<- %s stream=%d len=%d" % (NAMES.get(ftype, hex(ftype)), sid, len(payload)), verbose=True)
                if ftype == T_WINDOW:
                    inc = struct.unpack("!I", payload[:4])[0]
                    with self.cond:
                        if sid == 0:
                            self.conn_credit += inc
                        elif sid in self.streams:
                            self.streams[sid].tx_credit += inc
                        self.cond.notify_all()
                    continue
                st = self.streams.get(sid)
                if st is None:
                    continue
                if ftype == T_OPENED:
                    if st.proto == 1:
                        ip, port, _ = parse_addr(payload, 0)
                        st.bound = (ip, port)
                    st.opened.set()
                elif ftype == T_OPEN_FAIL:
                    st.fail = payload[0] if payload else 0x01
                    st.opened.set()
                    self.release(st)
                elif ftype == T_DATA:
                    if not st.opened.is_set():
                        self.close("DATA before OPENED", code=1002)
                        return
                    st.rx.put(payload)
                elif ftype == T_EOF:
                    st.rx.put(b"")
                elif ftype == T_RST:
                    log("stream %d RST reason=%d" % (sid, payload[0] if payload else -1), verbose=True)
                    st.rx.put(None)
                    self.release(st)
                else:
                    self.close("unexpected frame 0x%02x" % ftype, code=1002)
                    return
        except SessionGone:
            pass
        except OSError as e:
            self.close("read: %s" % e)
        finally:
            self.close("reader exit")

    def pinger(self):
        while not self.closed.wait(20.0):
            if time.monotonic() - self.pong_at > 60.0:
                self.close("3 pings unanswered", code=1001)
                return
            try:
                self.send_raw(0x9, b"kvm")
            except SessionGone:
                return

    def open(self, proto, ip=None, port=None):
        with self.cond:
            if len(self.streams) >= MAX_STREAMS or self.closed.is_set():
                return None
            sid, self.next_sid = self.next_sid, self.next_sid + 2
            st = KStream(sid, proto)
            self.streams[sid] = st
        payload = bytes([proto]) + (pack_addr(ip, port) if proto == 1 else b"")
        self.send(T_OPEN, sid, payload)
        return st

    def send_data(self, st, data):
        """Credit-gated DATA on a TCP stream; blocks until credit or session end."""
        off = 0
        while off < len(data):
            with self.cond:
                while not self.closed.is_set() and not st.released and (st.tx_credit <= 0 or self.conn_credit <= 0):
                    self.cond.wait(1.0)
                if self.closed.is_set() or st.released:
                    raise SessionGone()
                n = min(MAX_PAYLOAD, st.tx_credit, self.conn_credit, len(data) - off)
                st.tx_credit -= n
                self.conn_credit -= n
            self.send(T_DATA, st.sid, data[off:off + n])
            off += n

    def consumed(self, st, n):
        st.rx_unacked += n
        self.conn_unacked += n
        if st.rx_unacked >= STREAM_WINDOW // 2 and not st.released:
            self.send(T_WINDOW, st.sid, struct.pack("!I", st.rx_unacked))
            st.rx_unacked = 0
        if self.conn_unacked >= CONN_WINDOW // 2:
            self.send(T_WINDOW, 0, struct.pack("!I", self.conn_unacked))
            self.conn_unacked = 0

    def release(self, st):
        with self.cond:
            st.released = True
            self.streams.pop(st.sid, None)
            self.cond.notify_all()

    def rst(self, st, reason=0x01):
        if not st.released and not self.closed.is_set():
            try:
                self.send(T_RST, st.sid, bytes([reason]))
            except SessionGone:
                pass
        self.release(st)


CURRENT = {"s": None}
CURRENT_LOCK = threading.Lock()


def current():
    with CURRENT_LOCK:
        s = CURRENT["s"]
    return s if s is not None and not s.closed.is_set() else None


def ws_server(listener, tls_ctx):
    while True:
        conn, peer = listener.accept()
        threading.Thread(target=ws_conn, args=(conn, peer, tls_ctx), daemon=True).start()


def ws_conn(conn, peer, tls_ctx):
    try:
        conn.settimeout(10.0)
        if tls_ctx is not None:
            conn = tls_ctx.wrap_socket(conn, server_side=True)
        buf = b""
        while b"\r\n\r\n" not in buf:
            chunk = conn.recv(4096)
            if not chunk:
                return
            buf += chunk
        head, rest = buf.split(b"\r\n\r\n", 1)
        lines = head.decode("iso-8859-1").split("\r\n")
        method, path = lines[0].split(" ")[0:2]
        hdr = {}
        for line in lines[1:]:
            if ":" in line:
                k, v = line.split(":", 1)
                hdr[k.strip().lower()] = v.strip()
        want = "/exit/%s/native" % ARGS.slot
        authed = method == "GET" and hdr.get("authorization") == "Bearer " + ARGS.token
        script = path[len("/exit/%s/" % ARGS.slot):] if path.startswith("/exit/%s/client." % ARGS.slot) else ""
        if authed and script in ("client.sh", "client.py", "client.pl", "client.ps1"):
            body = render_client(script, hdr.get("host", ARGS.listen))
            conn.sendall(("HTTP/1.1 200 OK\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: %d\r\n"
                          "Connection: close\r\n\r\n" % len(body)).encode() + body)
            conn.close()
            log("served %s to %s" % (script, peer))
            return
        if (not authed or path != want
                or hdr.get("upgrade", "").lower() != "websocket" or "sec-websocket-key" not in hdr):
            log("rejecting %s %s from %s (auth=%r)" % (method, path, peer, hdr.get("authorization")))
            conn.sendall(b"HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\nContent-Length: 18\r\n\r\n404 page not found")
            conn.close()
            return
        accept = base64.b64encode(hashlib.sha1(hdr["sec-websocket-key"].encode() + WS_GUID).digest()).decode()
        conn.sendall(("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"
                      "Sec-WebSocket-Accept: %s\r\n\r\n" % accept).encode())
        if rest:
            log("client sent %d bytes before the 101 arrived; unsupported in the mock" % len(rest))
            conn.close()
            return
        sess = Session(conn, "%s:%d" % peer[:2])
        sess.handshake()
        with CURRENT_LOCK:
            prev, CURRENT["s"] = CURRENT["s"], sess
        if prev is not None and not prev.closed.is_set():
            log("superseding session from %s with %s" % (prev.peer, sess.peer))
            prev.close("superseded by a new exit", code=1000)
        log("exit attached from %s (%s, %s)" % (sess.peer, sess.hostname, sess.os))
        threading.Thread(target=sess.pinger, daemon=True).start()
        sess.reader()
    except (SessionGone, OSError, ssl.SSLError, ValueError, IndexError) as e:
        log("ws connection from %s ended: %s" % (peer, e))
        try:
            conn.close()
        except OSError:
            pass


# SOCKS5 front door ----------------------------------------------------------------

def socks_server(listener):
    while True:
        conn, _ = listener.accept()
        threading.Thread(target=socks_conn, args=(conn,), daemon=True).start()


def socks_reply(conn, rep, ip=ipaddress.IPv4Address("0.0.0.0"), port=0):
    conn.sendall(b"\x05" + bytes([rep]) + b"\x00" + pack_addr(ip, port))


def recv_exact(conn, n):
    buf = b""
    while len(buf) < n:
        c = conn.recv(n - len(buf))
        if not c:
            raise OSError("socks client closed")
        buf += c
    return buf


def socks_conn(conn):
    try:
        conn.settimeout(10.0)
        ver, nm = recv_exact(conn, 2)
        methods = recv_exact(conn, nm)
        if ver != 5 or 0 not in methods:
            conn.sendall(b"\x05\xff")
            return
        conn.sendall(b"\x05\x00")
        ver, cmd, _, atyp = recv_exact(conn, 4)
        if atyp == 1:
            ip = ipaddress.IPv4Address(recv_exact(conn, 4))
        elif atyp == 4:
            ip = ipaddress.IPv6Address(recv_exact(conn, 16))
        elif atyp == 3:
            n = recv_exact(conn, 1)[0]
            name = recv_exact(conn, n).decode("idna")
            ip = None
        else:
            socks_reply(conn, 0x08)
            return
        port = struct.unpack("!H", recv_exact(conn, 2))[0]
        if ip is None:
            ip = resolve(name, port)
            if ip is None:
                log("socks: cannot resolve %r" % name)
                socks_reply(conn, 0x04)
                return
        sess = current()
        if sess is None:
            socks_reply(conn, 0x03)
            return
        if cmd == 1:
            socks_connect(conn, sess, ip, port)
        elif cmd == 3:
            socks_udp(conn, sess)
        else:
            socks_reply(conn, 0x07)
    except (OSError, ValueError, struct.error, SessionGone) as e:
        log("socks connection ended: %s" % e, verbose=True)
    finally:
        try:
            conn.close()
        except OSError:
            pass


def socks_connect(conn, sess, ip, port):
    if not policy_allows(ip):
        socks_reply(conn, 0x02)
        return
    st = sess.open(1, ip, port)
    if st is None:
        socks_reply(conn, 0x01)
        return
    if not st.opened.wait(8.0):
        log("stream %d: OPEN timer expired" % st.sid)
        sess.rst(st, 0x06)
        socks_reply(conn, 0x06)
        return
    if st.fail is not None:
        log("stream %d: OPEN_FAIL reason=%d for %s:%d" % (st.sid, st.fail, ip, port))
        socks_reply(conn, st.fail)
        return
    socks_reply(conn, 0x00, st.bound[0], st.bound[1])
    conn.settimeout(None)
    done = threading.Event()

    def upstream():  # socks client -> exit
        try:
            while True:
                data = conn.recv(MAX_PAYLOAD)
                if not data:
                    sess.send(T_EOF, st.sid)
                    return
                sess.send_data(st, data)
        except (OSError, SessionGone):
            sess.rst(st, 0x01)
            done.set()

    t = threading.Thread(target=upstream, daemon=True)
    t.start()
    eof_from_exit = False
    try:
        while not done.is_set():
            item = st.rx.get()
            if item is None:
                break
            if item == b"":
                eof_from_exit = True
                try:
                    conn.shutdown(socket.SHUT_WR)
                except OSError:
                    pass
                continue
            conn.sendall(item)
            sess.consumed(st, len(item))
    except (OSError, SessionGone):
        sess.rst(st, 0x01)
    finally:
        # let the upstream side finish delivering the client's EOF, then drop the socket
        if eof_from_exit:
            t.join(30.0)
        try:
            conn.close()
        except OSError:
            pass
        sess.release(st)


def socks_udp(conn, sess):
    relay = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    relay.bind(("127.0.0.1", 0))
    st = sess.open(2)
    if st is None or not st.opened.wait(8.0) or st.fail is not None:
        socks_reply(conn, st.fail if st is not None and st.fail else 0x01)
        relay.close()
        return
    socks_reply(conn, 0x00, ipaddress.IPv4Address("127.0.0.1"), relay.getsockname()[1])
    client = {"addr": None}
    stop = threading.Event()

    def from_hev():
        relay.settimeout(1.0)
        idle = time.monotonic()
        while not stop.is_set():
            try:
                data, src = relay.recvfrom(65535)
            except socket.timeout:
                if time.monotonic() - idle > 60.0:
                    log("udp association idle 60 s; tearing down")
                    stop.set()
                continue
            except OSError:
                return
            idle = time.monotonic()
            if client["addr"] is None:
                client["addr"] = src
            elif src != client["addr"]:
                continue
            if len(data) < 4 or data[2] != 0:
                continue
            atyp = data[3]
            try:
                if atyp == 3:
                    n = data[4]
                    name = data[5:5 + n].decode("idna")
                    port = struct.unpack("!H", data[5 + n:7 + n])[0]
                    off = 7 + n
                    ip = resolve(name, port)
                    if ip is None:
                        continue
                else:
                    ip, port, off = parse_addr(data, 3)
            except (ValueError, IndexError, struct.error):
                continue
            if not policy_allows(ip):
                continue
            try:
                sess.send(T_DATA, st.sid, pack_addr(ip, port) + data[off:])
            except SessionGone:
                stop.set()

    def to_hev():
        while not stop.is_set():
            try:
                item = st.rx.get(timeout=1.0)
            except queue.Empty:
                continue
            if item is None or item == b"":
                stop.set()
                return
            try:
                ip, port, off = parse_addr(item, 0)
            except (ValueError, IndexError, struct.error):
                continue
            if client["addr"] is not None:
                relay.sendto(b"\x00\x00\x00" + pack_addr(ip, port) + item[off:], client["addr"])

    threads = [threading.Thread(target=from_hev, daemon=True), threading.Thread(target=to_hev, daemon=True)]
    for t in threads:
        t.start()
    conn.settimeout(1.0)
    try:
        while not stop.is_set():
            try:
                if not conn.recv(1):
                    break
            except socket.timeout:
                continue
    except OSError:
        pass
    stop.set()
    sess.rst(st, 0x01)
    relay.close()


TLS_FINGERPRINT = ""


def render_client(name, host):
    """Template a client the way commands.go does: ws/wss for the clients, http/https for the bootstrap."""
    with open(os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", name), "rb") as f:
        text = f.read().decode("utf-8")
    if name == "client.sh":
        scheme = "https" if ARGS.tls else "http"
    else:
        scheme = "wss" if ARGS.tls else "ws"
    for k, v in (("__SCHEME__", scheme), ("__HOST__", host), ("__SLOT__", ARGS.slot), ("__TOKEN__", ARGS.token),
                 ("__FINGERPRINT__", TLS_FINGERPRINT), ("__ALLOW_PRIVATE__", "1" if ARGS.allow_private else "0")):
        text = text.replace(k, v)
    return text.encode("utf-8")


def make_tls_context():
    d = tempfile.mkdtemp(prefix="mock_kvm_")
    crt, key = os.path.join(d, "crt.pem"), os.path.join(d, "key.pem")
    subprocess.check_call(["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-keyout", key,
                           "-out", crt, "-days", "2", "-subj", "/CN=mock-nanokvm"],
                          stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    ctx.load_cert_chain(crt, key)
    with open(crt) as f:
        der = ssl.PEM_cert_to_DER_cert(f.read())
    return ctx, hashlib.sha256(der).hexdigest()


def main():
    global ARGS, TLS_FINGERPRINT
    ap = argparse.ArgumentParser()
    ap.add_argument("--listen", default="127.0.0.1:0")
    ap.add_argument("--socks", default="127.0.0.1:0")
    ap.add_argument("--token", default="testtoken")
    ap.add_argument("--slot", default="0")
    ap.add_argument("--tls", action="store_true")
    ap.add_argument("--allow-private", action="store_true")
    ap.add_argument("-v", "--verbose", action="store_true")
    ARGS = ap.parse_args()
    tls_ctx = None
    if ARGS.tls:
        tls_ctx, fp = make_tls_context()
        TLS_FINGERPRINT = fp
        log("tls fingerprint sha256=%s" % fp)
        print("FINGERPRINT=%s" % fp, flush=True)
    ws = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    ws.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    h, p = ARGS.listen.rsplit(":", 1)
    ws.bind((h, int(p)))
    ws.listen(16)
    fd = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    fd.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    h, p = ARGS.socks.rsplit(":", 1)
    fd.bind((h, int(p)))
    fd.listen(64)
    log("listening %s://%s:%d/exit/%s/native (token=%s, allowPrivate=%s)" % (
        "wss" if ARGS.tls else "ws", ws.getsockname()[0], ws.getsockname()[1], ARGS.slot, ARGS.token, ARGS.allow_private))
    log("socks5 front door on %s:%d" % fd.getsockname())
    print("WS_PORT=%d" % ws.getsockname()[1], flush=True)
    print("SOCKS_PORT=%d" % fd.getsockname()[1], flush=True)
    threading.Thread(target=ws_server, args=(ws, tls_ctx), daemon=True).start()
    threading.Thread(target=socks_server, args=(fd,), daemon=True).start()
    try:
        while True:
            time.sleep(3600)
    except KeyboardInterrupt:
        return 0


if __name__ == "__main__":
    sys.exit(main())

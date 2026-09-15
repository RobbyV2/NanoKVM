#!/usr/bin/env python3
"""host_tests.py: drive a nexit/1 client through mock_kvm.py's SOCKS5 front door.

Starts throwaway TCP and UDP servers on --bind (a non-loopback address of this
host, because the destination policy always denies 127/8; run the client with
__ALLOW_PRIVATE__=1 and the mock with --allow-private when --bind is RFC 1918)
and exercises, through --socks:

  tcp-echo        1 MiB both ways, compared byte for byte
  tcp-bulk        10 MiB download, sha256 compared, throughput printed
  half-close      send, shutdown(SHUT_WR), read the reply until EOF (both EOF directions)
  open-fail       connect to a closed port -> SOCKS rep 0x05 (connection refused)
  open-timeout    connect to a black-holed address -> rep 0x06/0x03/0x04 within 9 s
  policy          with --expect-policy-deny: connect to --bind -> rep 0x02
  concurrent      20 parallel 256 KiB echoes
  udp-echo        5 datagrams through UDP ASSOCIATE, echoed back with the source recorded
  dns             a real A query for example.com to 1.1.1.1:53 via UDP ASSOCIATE

Exit status 0 when every selected test passes.
"""
import argparse
import hashlib
import os
import socket
import struct
import sys
import threading
import time


def socks_connect(socks, dst_ip, dst_port, timeout=15.0):
    """Returns (sock, rep). rep==0 on success."""
    s = socket.create_connection(socks, timeout=timeout)
    s.sendall(b"\x05\x01\x00")
    assert s.recv(2) == b"\x05\x00", "socks greeting"
    s.sendall(b"\x05\x01\x00\x01" + socket.inet_aton(dst_ip) + struct.pack("!H", dst_port))
    rep = recv_exact(s, 4)
    if rep[3] == 1:
        recv_exact(s, 6)
    elif rep[3] == 4:
        recv_exact(s, 18)
    if rep[1] != 0:
        s.close()
        return None, rep[1]
    s.settimeout(timeout)
    return s, 0


def socks_udp_associate(socks):
    ctl = socket.create_connection(socks, timeout=10)
    ctl.sendall(b"\x05\x01\x00")
    assert ctl.recv(2) == b"\x05\x00"
    ctl.sendall(b"\x05\x03\x00\x01" + b"\x00\x00\x00\x00" + b"\x00\x00")
    rep = recv_exact(ctl, 4)
    assert rep[1] == 0, "udp associate rep %d" % rep[1]
    bnd = recv_exact(ctl, 6)
    relay = (socket.inet_ntoa(bnd[:4]), struct.unpack("!H", bnd[4:])[0])
    u = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    u.bind(("127.0.0.1", 0))
    u.settimeout(5.0)
    return ctl, u, relay


def udp_wrap(ip, port, data):
    return b"\x00\x00\x00\x01" + socket.inet_aton(ip) + struct.pack("!H", port) + data


def udp_unwrap(pkt):
    assert pkt[:3] == b"\x00\x00\x00" and pkt[3] == 1, "udp header"
    return socket.inet_ntoa(pkt[4:8]), struct.unpack("!H", pkt[8:10])[0], pkt[10:]


def recv_exact(s, n):
    buf = b""
    while len(buf) < n:
        c = s.recv(n - len(buf))
        if not c:
            raise EOFError("short read: %d of %d" % (len(buf), n))
        buf += c
    return buf


def recv_all(s):
    out = bytearray()
    while True:
        c = s.recv(65536)
        if not c:
            return bytes(out)
        out += c


# --- local servers -----------------------------------------------------------

def tcp_server(bind, handler):
    ls = socket.socket()
    ls.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    ls.bind((bind, 0))
    ls.listen(64)

    def loop():
        while True:
            try:
                c, _ = ls.accept()
            except OSError:
                return
            threading.Thread(target=handler, args=(c,), daemon=True).start()

    threading.Thread(target=loop, daemon=True).start()
    return ls.getsockname()[1]


def echo_handler(c):
    try:
        while True:
            d = c.recv(65536)
            if not d:
                break
            c.sendall(d)
    except OSError:
        pass
    finally:
        c.close()


BULK = os.urandom(1 << 20) * 10
BULK_SHA = hashlib.sha256(BULK).hexdigest()


def bulk_handler(c):
    try:
        c.sendall(BULK)
    except OSError:
        pass
    finally:
        c.close()


def reverse_on_eof_handler(c):
    """Read until the client half-closes, then send everything back reversed and close."""
    try:
        data = recv_all(c)
        c.sendall(data[::-1])
    except OSError:
        pass
    finally:
        c.close()


def udp_echo_server(bind):
    u = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    u.bind((bind, 0))

    def loop():
        while True:
            try:
                d, src = u.recvfrom(65535)
                u.sendto(b"echo:" + d, src)
            except OSError:
                return

    threading.Thread(target=loop, daemon=True).start()
    return u.getsockname()[1]


# --- tests -------------------------------------------------------------------

def t_tcp_echo(a):
    port = tcp_server(a.bind, echo_handler)
    s, rep = socks_connect(a.socks, a.bind, port)
    assert rep == 0, "rep %d" % rep
    payload = os.urandom(1 << 20)
    got = bytearray()

    def reader():
        while len(got) < len(payload):
            c = s.recv(65536)
            if not c:
                break
            got.extend(c)

    t = threading.Thread(target=reader)
    t.start()
    s.sendall(payload)
    t.join(30)
    s.close()
    assert bytes(got) == payload, "echo mismatch (%d bytes)" % len(got)
    return "1 MiB echoed"


def t_tcp_bulk(a):
    port = tcp_server(a.bind, bulk_handler)
    s, rep = socks_connect(a.socks, a.bind, port, timeout=60)
    assert rep == 0, "rep %d" % rep
    t0 = time.monotonic()
    data = recv_all(s)
    dt = time.monotonic() - t0
    s.close()
    assert len(data) == len(BULK), "got %d bytes" % len(data)
    assert hashlib.sha256(data).hexdigest() == BULK_SHA, "sha256 mismatch"
    return "10 MiB in %.2fs (%.1f MB/s), sha256 ok" % (dt, len(data) / dt / 1e6)


def t_half_close(a):
    port = tcp_server(a.bind, reverse_on_eof_handler)
    s, rep = socks_connect(a.socks, a.bind, port)
    assert rep == 0, "rep %d" % rep
    payload = os.urandom(300 * 1024)
    s.sendall(payload)
    s.shutdown(socket.SHUT_WR)          # exit must forward EOF, not close
    data = recv_all(s)                  # server answers only after seeing our EOF
    s.close()
    assert data == payload[::-1], "reply mismatch (%d bytes)" % len(data)
    return "300 KiB sent, EOF forwarded, %d bytes reversed back, server EOF forwarded" % len(data)


def t_open_fail(a):
    ls = socket.socket()
    ls.bind((a.bind, 0))
    port = ls.getsockname()[1]
    ls.close()                           # nothing listens here now
    s, rep = socks_connect(a.socks, a.bind, port)
    if s:
        s.close()
    assert rep == 0x05, "expected rep 0x05, got 0x%02x" % rep
    return "closed port -> rep 0x05 connection refused"


def t_open_timeout(a):
    t0 = time.monotonic()
    s, rep = socks_connect(a.socks, a.blackhole, 9, timeout=20)
    dt = time.monotonic() - t0
    if s:
        s.close()
    assert rep in (0x03, 0x04, 0x06), "expected 0x03/0x04/0x06, got 0x%02x" % rep
    assert dt < 9.5, "took %.1fs" % dt
    return "%s:9 -> rep 0x%02x after %.1fs" % (a.blackhole, rep, dt)


def t_policy(a):
    port = tcp_server(a.bind, echo_handler)
    s, rep = socks_connect(a.socks, a.bind, port)
    if s:
        s.close()
    assert rep == 0x02, "expected rep 0x02, got 0x%02x" % rep
    return "private destination -> rep 0x02 not allowed by ruleset"


def t_concurrent(a):
    port = tcp_server(a.bind, echo_handler)
    errors = []

    def one(i):
        try:
            s, rep = socks_connect(a.socks, a.bind, port, timeout=60)
            assert rep == 0, "rep %d" % rep
            payload = os.urandom(256 * 1024)
            got = bytearray()

            def rd():
                while len(got) < len(payload):
                    c = s.recv(65536)
                    if not c:
                        break
                    got.extend(c)

            t = threading.Thread(target=rd)
            t.start()
            s.sendall(payload)
            t.join(60)
            s.close()
            assert bytes(got) == payload, "mismatch"
        except Exception as e:  # noqa
            errors.append("%d: %s" % (i, e))

    ts = [threading.Thread(target=one, args=(i,)) for i in range(20)]
    for t in ts:
        t.start()
    for t in ts:
        t.join(90)
    assert not errors, errors[:3]
    return "20 x 256 KiB echoes in parallel"


def t_udp_echo(a):
    port = udp_echo_server(a.bind)
    ctl, u, relay = socks_udp_associate(a.socks)
    ok = 0
    for i in range(5):
        msg = ("hello %d " % i).encode() + os.urandom(100)
        u.sendto(udp_wrap(a.bind, port, msg), relay)
        pkt, _ = u.recvfrom(65535)
        src_ip, src_port, data = udp_unwrap(pkt)
        assert data == b"echo:" + msg, "datagram %d mismatch" % i
        assert (src_ip, src_port) == (a.bind, port), "source %s:%d" % (src_ip, src_port)
        ok += 1
    u.close()
    ctl.close()
    return "%d datagrams echoed, reply source %s:%d recorded" % (ok, a.bind, port)


def t_dns(a):
    ctl, u, relay = socks_udp_associate(a.socks)
    qid = os.urandom(2)
    q = qid + b"\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00" + b"\x07example\x03com\x00" + b"\x00\x01\x00\x01"
    u.sendto(udp_wrap("1.1.1.1", 53, q), relay)
    pkt, _ = u.recvfrom(65535)
    src_ip, src_port, data = udp_unwrap(pkt)
    u.close()
    ctl.close()
    assert (src_ip, src_port) == ("1.1.1.1", 53), "source %s:%d" % (src_ip, src_port)
    assert data[:2] == qid, "dns id"
    ancount = struct.unpack("!H", data[6:8])[0]
    assert ancount >= 1, "no answers"
    return "A example.com via 1.1.1.1:53 -> %d answer(s)" % ancount


def t_slow_reader(a):
    """Download 10 MiB but do not read for 2 s: exit->kvm credit must run out and resume."""
    port = tcp_server(a.bind, bulk_handler)
    s, rep = socks_connect(a.socks, a.bind, port, timeout=60)
    assert rep == 0, "rep %d" % rep
    time.sleep(2.0)
    data = bytearray()
    while len(data) < len(BULK):
        c = s.recv(65536)
        if not c:
            break
        data += c
        if len(data) < 2 << 20:
            time.sleep(0.005)          # trickle for the first 2 MiB, then drain
    s.close()
    assert len(data) == len(BULK), "got %d bytes" % len(data)
    assert hashlib.sha256(bytes(data)).hexdigest() == BULK_SHA, "sha256 mismatch"
    return "10 MiB with a stalled then trickling reader, sha256 ok"


def slow_sink_handler(c):
    """Read slowly, then answer with the sha256 of everything received."""
    h = hashlib.sha256()
    total = 0
    try:
        while True:
            d = c.recv(16384)
            if not d:
                break
            h.update(d)
            total += len(d)
            if total < 1 << 20:
                time.sleep(0.01)
        c.sendall(h.hexdigest().encode())
    except OSError:
        pass
    finally:
        c.close()


def t_slow_server(a):
    """Upload 8 MiB into a slow reader: kvm->exit credit must run out and resume."""
    port = tcp_server(a.bind, slow_sink_handler)
    s, rep = socks_connect(a.socks, a.bind, port, timeout=120)
    assert rep == 0, "rep %d" % rep
    payload = os.urandom(1 << 20) * 8
    t0 = time.monotonic()
    s.sendall(payload)
    s.shutdown(socket.SHUT_WR)
    digest = recv_all(s).decode()
    s.close()
    assert digest == hashlib.sha256(payload).hexdigest(), "server digest mismatch"
    return "8 MiB into a slow sink in %.1fs, digest ok" % (time.monotonic() - t0)


TESTS = [("tcp-echo", t_tcp_echo), ("tcp-bulk", t_tcp_bulk), ("half-close", t_half_close),
         ("open-fail", t_open_fail), ("open-timeout", t_open_timeout), ("concurrent", t_concurrent),
         ("slow-reader", t_slow_reader), ("slow-server", t_slow_server),
         ("udp-echo", t_udp_echo), ("dns", t_dns)]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--socks", default="127.0.0.1:18081")
    ap.add_argument("--bind", required=True, help="non-loopback address of this host for the test servers")
    ap.add_argument("--blackhole", default="10.255.255.1", help="address that drops SYNs")
    ap.add_argument("--only", default="", help="comma-separated test names")
    ap.add_argument("--expect-policy-deny", action="store_true", help="run only the policy test")
    a = ap.parse_args()
    h, p = a.socks.rsplit(":", 1)
    a.socks = (h, int(p))
    tests = TESTS
    if a.expect_policy_deny:
        tests = [("policy", t_policy)]
    elif a.only:
        want = set(a.only.split(","))
        tests = [t for t in TESTS if t[0] in want]
    failed = 0
    for name, fn in tests:
        t0 = time.monotonic()
        try:
            msg = fn(a)
            print("PASS %-13s %s (%.2fs)" % (name, msg, time.monotonic() - t0))
        except Exception as e:  # noqa
            failed += 1
            print("FAIL %-13s %s: %s (%.2fs)" % (name, type(e).__name__, e, time.monotonic() - t0))
        sys.stdout.flush()
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())

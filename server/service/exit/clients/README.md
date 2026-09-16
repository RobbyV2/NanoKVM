# nexit/1 exit clients

The exit-device half of Mode A (spec: `docs/superpowers/specs/2026-09-15-exit-tunnel-design.md`,
section "The nexit/1 protocol (Mode A)" and "Client architecture"; plan: W3 clients). Each
client is one file with no dependency beyond its runtime, served fully templated by the
NanoKVM from `GET /exit/<slot>/client.<ext>` with `Authorization: Bearer <token>`, and takes
no arguments.

| File | Runtime | Notes |
|---|---|---|
| `client.py` | Python 3.8+, stdlib | `selectors` over one non-blocking socket or `ssl.SSLSocket` plus stream sockets |
| `client.pl` | Perl 5.18+ core; `IO::Socket::SSL` only for `wss` | `IO::Select`, raw `socket`/`connect` for streams, `IO::Socket::IP` (falls back to `IO::Socket::INET`) for the WebSocket |
| `client.ps1` | Windows PowerShell 5.1, .NET Framework 4.5+ | `ClientWebSocket`, `Task.WaitAny` polling, compiled `Add-Type` pin callback |
| `client.sh` | POSIX sh + curl (+ openssl on https) | D14 bootstrap: picks python3 or perl, fetches that client, runs it |
| `testdata/mock_kvm.py` | Python 3 | the kvm side of nexit/1 plus a SOCKS5 front door, for host testing without Go |
| `testdata/host_tests.py` | Python 3 | drives TCP/UDP/half-close/flow-control/policy tests through the mock's front door |
| `testdata/windows-checklist.md` | | how to exercise `client.ps1` on Windows PowerShell 5.1 itself, which the `pwsh` run below does not cover |

## Templating

Placeholders replaced by `commands.go` when serving (never argv):

| Placeholder | client.py / .pl / .ps1 | client.sh |
|---|---|---|
| `__SCHEME__` | `ws` or `wss` (the clients also accept `http`/`https`) | `http` or `https` (also accepts `ws`/`wss`) |
| `__HOST__` | `host[:port]` as the operator reached the UI, IPv6 in brackets | same |
| `__SLOT__` | `0` | same |
| `__TOKEN__` | the slot token | same |
| `__FINGERPRINT__` | sha256 hex of the served leaf certificate, or empty on http; colons and upper case are tolerated | same |
| `__ALLOW_PRIVATE__` | `1` or `0` | not used |

Every client refuses to run with the placeholders still in place. `clients_embed.go` embeds
exactly the four `client.*` files, so this README and `testdata/` stay out of the binary.

The Go conformance suite (`server/service/exit/conformance_test.go`, build tag `conformance`)
runs `client.py` and `client.pl` as subprocesses against the real Mux and FrontDoor:

    cd server && go test -race -tags conformance -run Conformance ./service/exit/ -v

It needs `python3`, `perl` and a non-loopback IPv4 address on the host (the policy always
denies 127/8, so the echo servers listen on the LAN address, as `host_tests.py --bind` does).

## Behaviour common to the three clients

- WebSocket: RFC 6455 client handshake with a random key and `Sec-WebSocket-Accept` check
  (Python and Perl do it by hand; .NET does it internally), masked client frames, fragment
  reassembly, read limit 16 KiB + 64 per message, exactly one nexit frame per binary message;
  a text message, a masked server frame or an oversized message ends the session.
- TLS: `wss` with a non-empty fingerprint verifies the leaf certificate's SHA-256 after the
  handshake and before any byte is trusted (Python `getpeercert(binary_form=True)`, Perl
  `get_fingerprint('sha256')`, PowerShell a compiled `RemoteCertificateValidationCallback`).
  `wss` with an empty fingerprint is refused: nothing is ever silently unverified. CA
  validation is off by design; the pin is the trust anchor (D14).
- HELLO first: `version=1, flags (bit 0: IPv4 default route present), hostname, os`, both
  strings with a one-byte length prefix. `os` is `darwin`, `linux` or `windows`. WELCOME must be
  the first frame back; its `streamWindow`/`connWindow`/`maxStreams` are honoured.
- Flow control: additive credit per TCP stream and per connection (stream 0); the client stops
  reading a socket when either credit is 0 and resumes on WINDOW; it sends WINDOW for a stream
  after passing half of `streamWindow` to the socket and for the connection after consuming half
  of `connWindow`; bytes buffered for a stream that gets RST are still returned to the connection
  window. UDP is uncredited; at most 32 unsent datagrams per stream are queued toward the kvm
  (oldest dropped) in Python and Perl; PowerShell never queues because every send is awaited.
- Streams: OPEN → OPENED with the bound address, or OPEN_FAIL with a reason; DATA on a stream
  not yet OPENED and OPEN on a live or even id are session errors; frames for unknown ids are
  ignored; DATA after the kvm's EOF is answered with RST; a stream is released after both EOFs
  or any RST.
- Connect: non-blocking with a 6 s budget (`connect_ex`+`SO_ERROR` / `connect`+`sockopt(SO_ERROR)`
  / `ConnectAsync` polled), reasons per the spec table (refused 5, host unreachable 4, network
  unreachable 3, timeout 6, address family 8, else 1). IPv6 destinations (atyp 4) are attempted
  when the host has an IPv6 socket; a failure maps to the same table.
- Half close: kvm EOF → pending bytes flushed → `shutdown(SHUT_WR)` (`shutdown($s,1)`,
  `Socket.Shutdown(Send)`); reading continues until the socket returns 0, then EOF is sent.
- Policy (D21) applied at OPEN and at UDP send, mirroring `server/service/exit/policy.go`;
  `server/service/exit/clients_test.go` asserts every prefix in `policy.go` appears in each client.
- Keepalive: WS ping every 30 s; 75 s without any frame ends the session (Python, Perl). The
  PowerShell client cannot see ping/pong frames through `ClientWebSocket`, so it uses
  `KeepAliveInterval` 30 s and treats a faulted receive or a non-Open state as the signal; the
  kvm's own 20 s ping / 3 misses still catches a dead exit.
- Reconnect loop: backoff 1, 2, 4, 8, 16, 30 s, reset to 1 after a session that reached WELCOME.
  A 404 is reported as "wrong token or slot, or this source is rate limited".
- UDP: one unconnected IPv4 socket per stream; every reply's source goes in the DATA header.
  IPv6 datagram destinations are dropped.

## client.sh

1. Picks the runtime: on Darwin python3 only if `xcode-select -p` succeeds and
   `import ssl, socket, selectors` works; on Linux python3 if that import works; otherwise perl.
2. Fetches `client.py` or `client.pl` with the token in the header. On https the verified fetch
   is tried first; if it fails with a TLS error and a fingerprint is templated, the served
   certificate's SHA-256 is read with `openssl s_client` and compared to the pin, and only on a
   match is the fetch repeated with `-k`. No pin or no openssl → loud failure.
3. Requires a non-empty body whose first three lines contain the language marker
   (`nexit/1 python client` / `nexit/1 perl client`), then `exec python3 -c "$body"` or
   `exec perl -e "$body"`.

## Test log

Host: macOS 26.6.2 (Darwin 25.6.0 arm64). Tools: `/usr/bin/python3` 3.9.6 (LibreSSL 2.8.3),
Homebrew `python3` 3.14.7 (OpenSSL 3.6.4; used by `client.sh` because it is first on PATH),
`/usr/bin/perl` 5.34.1 with IO::Socket::SSL 2.068 / Net::SSLeay 1.88 (LibreSSL 3.3.6),
curl 8.7.1, openssl 3.6.4 (mock certificate generation), shellcheck 0.11.0, go 1.27.1.
LAN address of the host: 10.36.213.238 (`ipconfig getifaddr en0`); the test servers must sit
on a non-loopback address because the policy always denies 127/8, so the mock ran with
`--allow-private` and the clients with `__ALLOW_PRIVATE__=1` except where noted.
Templating for the direct runs was a `sed` of the six placeholders.

### Python and Perl, ws

```
python3 testdata/mock_kvm.py --listen 127.0.0.1:18080 --socks 127.0.0.1:18081 --token tok --allow-private
python3 c.py      # client.py templated ws 127.0.0.1:18080 slot 0 token tok fp "" allowPrivate 1
```
Client: `connected: streamWindow=131072 connWindow=4194304 maxStreams=256`; mock:
`HELLO version=1 flags=0x1 hostname='dhanya-10.local' os='darwin'`, `exit attached`.

```
curl -sS --socks5-hostname 127.0.0.1:18081 -o /dev/null -w '%{http_code} %{size_download}' http://example.com/
  python: 200 559 (0.044 s)          perl: 200 559 (0.024 s)
```

10 MB transfer, `python3 -m http.server --bind 10.36.213.238 18090` serving a random file
(sha256 8b4fbcba…698f):
```
curl -sS --socks5-hostname 127.0.0.1:18081 -o via_socks.bin http://10.36.213.238:18090/10mb.bin
  python: 200 10485760 bytes in 0.038 s, sha256 8b4fbcba…698f (match)
  perl:   200 10485760 bytes in 0.031 s, sha256 8b4fbcba…698f (match)
```

`python3 testdata/host_tests.py --socks 127.0.0.1:18081 --bind 10.36.213.238` (all PASS for both):
```
                python client                              perl client
tcp-echo        1 MiB echoed                               1 MiB echoed
tcp-bulk        10 MiB in 0.04 s (298 MB/s), sha256 ok     10 MiB in 0.03 s (359 MB/s), sha256 ok
half-close      300 KiB, EOF forwarded, reversed back      same
open-fail       closed port -> rep 0x05                    same
open-timeout    10.255.255.1:9 -> rep 0x06 after 6.0 s     same
concurrent      20 x 256 KiB echoes in parallel            same
slow-reader     10 MiB, reader stalled 2 s then trickling  same (2.3 s)
slow-server     8 MiB into a slow sink in 0.9 s            same
stalled-sink    256 KiB echoed in 0.01 s beside a wedged   same (0.03 s)
                1 MiB upload into a sink that never reads
udp-echo        5 datagrams, reply source recorded         same
dns             A example.com via 1.1.1.1:53 -> 2 answers  same
```

Policy (client templated with `__ALLOW_PRIVATE__=0`, mock still `--allow-private` so the
refusal is the client's):
```
python3 testdata/host_tests.py ... --expect-policy-deny
  python: PASS policy  private destination -> rep 0x02       perl: PASS policy  rep 0x02
curl --socks5-hostname 127.0.0.1:18081 http://10.36.213.238:18090/10mb.bin
  curl: (97) Can't complete SOCKS5 connection to 10.36.213.238. (2)
```

`nc` half-close style (`printf 'GET / HTTP/1.0\r\nHost: example.com\r\n\r\n' | nc -X 5 -x 127.0.0.1:18081 example.com 80`):
macOS `nc` exits on stdin EOF before reading the reply even without a proxy (0 bytes in both
cases), so it cannot demonstrate half-close on this host. With stdin held open
(`(printf ...; sleep 3) | nc -X 5 -x ...`) the tunnelled reply is 828 bytes, identical to the
direct connection. Half close proper is covered by the `half-close` test above (client sends,
`shutdown(SHUT_WR)`, server answers only after seeing EOF, then closes; both EOF directions).

Reconnect and supersede (`kill` the mock, wait, restart; then a second client):
```
python: session ended: websocket closed by peer / reconnecting in 1s / 2s / 4s / 8s / connected
perl:   same sequence, connected after the 8 s step; tcp-echo PASS afterwards
second client attaches -> mock: superseding session from :50009 with :50011
first client: session ended: websocket closed by kvm (1000 superseded by a new exit) / reconnecting in 1s / connected
```

Idle keepalive (mock pings every 20 s and closes after 60 s without a pong; client pings every
30 s and drops after 75 s without a frame): 85 s idle after connect, no `session ended`,
then `tcp-echo` PASS — python (94 s session) and perl (94 s session).

### Python and Perl, wss with a self-signed certificate

```
python3 testdata/mock_kvm.py --listen 127.0.0.1:18443 --socks 127.0.0.1:18081 --token tok --tls --allow-private
  tls fingerprint sha256=a3e1e95b86eb5243712d7b9e1c82cd52d69f73be0b46148ff096ada40355d076
```
Both clients, templated `wss`:
- correct pin: `connected`; tcp-echo, tcp-bulk (199–221 MB/s), half-close, udp-echo PASS
- wrong pin: `connect failed: certificate fingerprint mismatch: expected b4f2… got a3e1…; refusing`, backoff 1, 2, 4 s, no session on the mock
- empty pin: `connect failed: wss without a certificate fingerprint: refusing to connect unverified`
- pin as `A3:E1:…` (colons, upper case): `connected`, tcp-echo PASS
- scheme templated as `https` instead of `wss`: `connected`, tcp-echo PASS
- wrong token: `connect failed: rejected (404): wrong token or slot, or this source is rate limited`; mock: `rejecting GET /exit/0/native ... (auth='Bearer badtoken')`

### client.sh

The mock serves `/exit/0/client.{sh,py,pl,ps1}` templated the way `commands.go` will, so the
real one-liner shape was run:
```
curl -fsS -H 'Authorization: Bearer tok' http://127.0.0.1:18080/exit/0/client.sh | sh
  -> python3 picked (xcode-select -p succeeds); process is `python3 -c <body>`; connected;
     tcp-echo, tcp-bulk (275 MB/s), half-close, udp-echo PASS
curl ... client.sh | env PATH=<dir with sh curl perl uname head grep sed tr openssl, no python3> sh
  -> perl picked; process is `perl -e <body>`; connected; same four tests PASS
curl -fsS -H 'Authorization: Bearer nope' .../client.sh | sh
  -> curl: (22) The requested URL returned error: 404 ; sh runs nothing (the first fetch is the one-liner's)
client.sh pointed at a server answering HTML
  -> nexit: unexpected response from http://127.0.0.1:18099/exit/0/client.py (not the nexit/1 python client); exit 1
curl -fsSk -H 'Authorization: Bearer tok' https://127.0.0.1:18443/exit/0/client.sh | sh   (mock --tls)
  -> verified fetch: curl: (60) SSL certificate problem: self signed certificate
     openssl s_client … | openssl x509 -fingerprint -sha256 matches the pin
     refetch with -k; wss client with pin: connected; tcp-echo, udp-echo PASS
same with FINGERPRINT edited to 000…0
  -> nexit: certificate fingerprint mismatch: expected 000…0 got c725…4011; refusing ; exit 1, no -k fetch
same with FINGERPRINT=""
  -> nexit: TLS verification failed (curl 60) and no certificate fingerprint is pinned; refusing ; exit 1
shellcheck client.sh -> clean
```

### Go

```
cd server && gofmt -l . && go vet ./service/exit/... && go test -race ./service/exit/...
  gofmt: nothing; vet ok; ok NanoKVM-Server/service/exit (TestClientsCarryPlaceholdersAndMarkers,
  TestClientsMirrorPolicyPrefixes); the mirror test was shown to fail when one prefix in
  client.ps1 was changed to 198.18.0.0/16, then restored.
```

### PowerShell

No Windows here. `client.ps1` was written against .NET Framework 4.5+ APIs available in
Windows PowerShell 5.1 and reviewed for the usual traps (array unrolling on return, `-shl` on
`[int]`, hashtable key types, `ArraySegment[byte]` construction, `Add-Type` re-definition, the
`$Host` automatic variable). It was then run under PowerShell 7 (`pwsh` 7.6.6, a `dotnet tool`)
on the same macOS host, templated with `sed` and started as `pwsh -NoProfile -File c.ps1`:

```
python3 testdata/mock_kvm.py --listen 127.0.0.1:18080 --socks 127.0.0.1:18081 --token tok --allow-private
  client: exit client for ws://127.0.0.1:18080/exit/0 (allowPrivate=True, pin=none)
          connected: streamWindow=131072 connWindow=4194304 maxStreams=256
  mock:   HELLO version=1 flags=0x1 hostname='dhanya-10.local' os='windows'
python3 testdata/host_tests.py --socks 127.0.0.1:18081 --bind 192.168.139.3   (all PASS)
  tcp-echo      1 MiB echoed
  tcp-bulk      10 MiB in 0.17 s (60.5 MB/s), sha256 ok
  half-close    300 KiB sent, EOF forwarded, reversed back, server EOF forwarded
  open-fail     closed port -> rep 0x05
  open-timeout  10.255.255.1:9 -> rep 0x06 after 6.1 s
  concurrent    20 x 256 KiB echoes in parallel
  slow-reader   10 MiB with a stalled then trickling reader, sha256 ok (2.3 s)
  slow-server   8 MiB into a slow sink in 0.9 s, digest ok
  stalled-sink  256 KiB echoed in 0.03 s while 1 MiB sat in a sink that never reads
  udp-echo      5 datagrams echoed, reply source recorded
  dns           A example.com via 1.1.1.1:53 -> 2 answers
```

`stalled-sink` was added for the head-of-line fix: the first `client.ps1` wrote kvm DATA to
the destination with a synchronous `NetworkStream.Write` (30 s `WriteTimeout`), so one
destination that stopped reading blocked the only thread and every other stream with it.
Against that version the test fails as predicted, `FAIL stalled-sink TimeoutError: timed out
(6.54s)` on the echo's SOCKS CONNECT with no `OPENED` from the client; with the per-stream
write queue and one polled `WriteAsync` it passes, and the sink stream holds one stream window.

The `pwsh` run covers the protocol and the task loop on .NET 10, not Windows PowerShell 5.1,
.NET Framework's `ClientWebSocket`, the `ServicePointManager` pin callback or Windows socket
buffer sizes. `testdata/windows-checklist.md` is the acceptance run for those.

## Re-running

```
python3 testdata/mock_kvm.py --listen 127.0.0.1:18080 --socks 127.0.0.1:18081 --token tok --allow-private [--tls] [-v]
curl -fsS -H 'Authorization: Bearer tok' http://127.0.0.1:18080/exit/0/client.sh | sh
python3 testdata/host_tests.py --socks 127.0.0.1:18081 --bind <LAN address of this host>
```

`-v` on the mock logs every nexit frame. Note that on macOS `pkill -f` cannot match a
process whose argv is a 30 KB script; kill the exec'd client by pid.

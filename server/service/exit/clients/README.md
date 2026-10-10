# nexit/1 exit clients

The exit-device half of Mode A. Each client is one file with no dependency beyond its runtime, served fully templated by the
NanoKVM from `GET /exit/<slot>/client.<ext>` with `Authorization: Bearer <token>`, and takes
no arguments.

| File | Runtime | Notes |
|---|---|---|
| `client.py` | Python 3.8+, stdlib | `selectors` over one non-blocking socket or `ssl.SSLSocket` plus stream sockets |
| `client.pl` | Perl 5.18+ core; `IO::Socket::SSL` only for `wss` | `IO::Select`, raw `socket`/`connect` for streams, `IO::Socket::IP` (falls back to `IO::Socket::INET`) for the WebSocket |
| `client.ps1` | Windows PowerShell 5.1, .NET Framework 4.5+ | `ClientWebSocket`, `Task.WaitAny` polling, compiled `Add-Type` pin callback |
| `client.sh` | POSIX sh + curl (+ openssl on https) | Bootstrap: picks python3 or perl, fetches that client, runs it |
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

## The nexit/1 protocol (Mode A)

Each binary WebSocket message contains one `type:u8 stream:u32be payload` frame,
with at most 16 KiB of payload. Stream 0 carries connection control;
`server/service/exit/frame.go` defines the frame types and encoding.

## Behaviour common to the three clients

- WebSocket: RFC 6455 client handshake with a random key and `Sec-WebSocket-Accept` check
  (Python and Perl do it by hand; .NET does it internally), masked client frames, fragment
  reassembly, read limit 16 KiB + 64 per message, exactly one nexit frame per binary message;
  a text message, a masked server frame or an oversized message ends the session.
- TLS: `wss` with a non-empty fingerprint verifies the leaf certificate's SHA-256 after the
  handshake and before any byte is trusted (Python `getpeercert(binary_form=True)`, Perl
  `get_fingerprint('sha256')`, PowerShell a compiled `RemoteCertificateValidationCallback`).
  `wss` with an empty fingerprint is refused: nothing is ever silently unverified. CA
  validation is off by design; the pin is the trust anchor.
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
  / `ConnectAsync` polled), reason codes (refused 5, host unreachable 4, network
  unreachable 3, timeout 6, address family 8, else 1). IPv6 destinations (atyp 4) are attempted
  when the host has an IPv6 socket; a failure maps to the same table.
- Half close: kvm EOF → pending bytes flushed → `shutdown(SHUT_WR)` (`shutdown($s,1)`,
  `Socket.Shutdown(Send)`); reading continues until the socket returns 0, then EOF is sent.
- Policy applied at OPEN and at UDP send, mirroring `server/service/exit/policy.go`;
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

## Host validation

```
python3 testdata/mock_kvm.py --listen 127.0.0.1:18080 --socks 127.0.0.1:18081 --token tok --allow-private [--tls] [-v]
curl -fsS -H 'Authorization: Bearer tok' http://127.0.0.1:18080/exit/0/client.sh | sh
python3 testdata/host_tests.py --socks 127.0.0.1:18081 --bind <LAN address of this host>
```

`-v` on the mock logs every nexit frame. Note that on macOS `pkill -f` cannot match a
process whose argv is a 30 KB script; kill the exec'd client by pid.

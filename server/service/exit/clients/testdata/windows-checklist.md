# client.ps1: manual checklist for Windows PowerShell 5.1

`client.ps1` targets Windows PowerShell 5.1 on .NET Framework, which is not available
where it was written. It has been executed under PowerShell 7 (`pwsh` 7.6.6, installed
as a `dotnet tool`) on macOS against `mock_kvm.py`, where every `host_tests.py` test
passes; see the README's test log. That run covers the protocol and the task loop but
not Windows PowerShell 5.1 itself, .NET Framework's `ClientWebSocket`, the
`ServicePointManager` pin callback or Windows socket buffer sizes. This checklist is how
those get exercised for real. Work through it once on Windows 10 or 11 before the first
live test, and again whenever `client.ps1` changes. Tick every box or file the failure
with the exact output.

## 0. Prerequisites on the Windows machine

- [ ] `$PSVersionTable.PSVersion` is 5.1.x (Windows PowerShell, not `pwsh`; `pwsh` 7 is a
      bonus run, not the target).
- [ ] `$ExecutionContext.SessionState.LanguageMode` is `FullLanguage` (Add-Type needs it).
- [ ] `Add-Type -TypeDefinition 'public static class T { public static int X = 1; }'; [T]::X`
      prints `1` (the C# compiler is available; AppLocker/WDAC may block it).
- [ ] `[Net.ServicePointManager]::SecurityProtocol` can be set to `'Tls12,Tls13'` on .NET 4.8;
      on an older .NET the client falls back to `Tls12`, which is acceptable.
- [ ] Windows 8 / Server 2012 or newer. On Windows 7, `New-Object Net.WebSockets.ClientWebSocket`
      throws `PlatformNotSupportedException`; the client is expected to fail there and the
      UI should say so.

## 1. A NanoKVM stand-in on the LAN

On a Mac or Linux box reachable from the Windows machine (address `LAN` below, must not
be 127.x; the destination policy always denies loopback):

```sh
python3 server/service/exit/clients/testdata/mock_kvm.py --listen LAN:18080 --socks 127.0.0.1:18081 --token tok --allow-private
```

`--allow-private` is needed because the throwaway test servers live on `LAN`, which is
RFC 1918; the mock templates `__ALLOW_PRIVATE__=1` into the served client accordingly.

## 2. Plain http (ws)

On Windows, in a fresh Windows PowerShell window:

```powershell
irm -Headers @{Authorization='Bearer tok'} http://LAN:18080/exit/0/client.ps1 | iex
```

- [ ] Output: `nexit: exit client for ws://LAN:18080/exit/0 (allowPrivate=True, pin=none)` then
      `nexit: connected: streamWindow=131072 connWindow=4194304 maxStreams=256`.
- [ ] The mock logs `HELLO version=1 flags=0x1 hostname='<PC name>' os='windows'` and
      `exit attached from ...`.

On the mock host:

```sh
python3 server/service/exit/clients/testdata/host_tests.py --socks 127.0.0.1:18081 --bind LAN
```

- [ ] `tcp-echo` PASS.
- [ ] `tcp-bulk` PASS; note the MB/s (5–20 MB/s is the expectation for the task-polled loop).
- [ ] `half-close` PASS (both EOF directions; exercises `Socket.Shutdown(Send)`).
- [ ] `open-fail` PASS (SocketErrorCode ConnectionRefused -> 0x05).
- [ ] `open-timeout` PASS within 9 s (the 6 s ConnectAsync budget -> 0x06; if Windows itself
      reports TimedOut earlier that also maps to 0x06).
- [ ] `concurrent` PASS (20 parallel streams; task list handling).
- [ ] `slow-reader` and `slow-server` PASS (credit exhaustion and WINDOW resumption both ways).
- [ ] `stalled-sink` PASS: 1 MiB is pushed into a destination that accepts and never reads
      while a 256 KiB echo on another stream must complete within 5 s. This is the
      head-of-line check: kvm DATA is queued per stream and written with one `WriteAsync`
      polled in the loop, so a wedged destination must not stop OPEN, other streams' DATA
      or WINDOW. The Windows send buffer default (64 KiB) makes the sink stream wedge
      sooner than on macOS; the echo timing must not change. A failure here shows as
      `TimeoutError` on the echo's CONNECT and `OPEN timed out` on the kvm.
- [ ] `udp-echo` PASS (UdpClient ReceiveAsync/Send; source recorded in the DATA header).
- [ ] `dns` PASS (a real A query to 1.1.1.1:53 through UDP ASSOCIATE).

Also from the mock host, a real tool:

```sh
curl --socks5-hostname 127.0.0.1:18081 -o /dev/null -w '%{http_code} %{size_download} %{speed_download}\n' http://example.com/
```

- [ ] `200 ...`.

## 3. https (wss) with the fingerprint pin

Restart the mock with `--tls` (it prints `FINGERPRINT=<sha256>`), then on Windows:

```powershell
if (-not ('NanoKVMExitTrust' -as [type])) { Add-Type -TypeDefinition 'using System.Net;using System.Net.Security;using System.Security.Cryptography.X509Certificates;public static class NanoKVMExitTrust{public static bool Ok(object s,X509Certificate c,X509Chain h,SslPolicyErrors e){return true;}public static void Install(){ServicePointManager.ServerCertificateValidationCallback=new RemoteCertificateValidationCallback(Ok);}}' }; [NanoKVMExitTrust]::Install(); irm -Headers @{Authorization='Bearer tok'} https://LAN:18443/exit/0/client.ps1 | iex
```

- [ ] `connected:` line appears; the mock logs the session; `host_tests.py --only tcp-echo,udp-echo` PASS.
- [ ] Ctrl+C, then `([Net.ServicePointManager]::ServerCertificateValidationCallback).Method.DeclaringType`
      prints `NanoKVMExitTrust`, not `NexitPin` (the client restores the previous callback on exit;
      the one-liner's compiled trust-all delegate, which the pin replaced during the run).
- [ ] Wrong pin: fetch the script to a file (`irm ... -OutFile c.ps1`), edit `$NexitFingerprint`
      to another 64-hex value, run `.\c.ps1`. Expect
      `connect failed: certificate fingerprint mismatch: expected ... got <served>; refusing`
      and reconnect attempts with 1, 2, 4 ... 30 s backoff, never `connected:`.
- [ ] Empty pin: set `$NexitFingerprint = ''`, run. Expect
      `wss without a certificate fingerprint: refusing to connect unverified`.
- [ ] The message `the certificate was never presented for pinning (connection reused); refusing`
      must NOT appear with the correct pin. If it does, .NET reused the `irm` connection for the
      WebSocket; report it, that is the fail-closed guard firing, and the fix is a different
      connection-group strategy in `Connect-Kvm`.
- [ ] Run the wss test twice in the same window (Ctrl+C in between). The second run must not
      fail on `Add-Type` ("type already exists" is guarded by the `-as [type]` check).

## 4. Lifecycle

- [ ] Ctrl+C in the client window: `nexit: stopping`; the mock logs
      `exit closed the websocket (1000 client exiting)`.
- [ ] Stop the mock while the client is connected: `session ended: websocket ...` then
      `reconnecting in 1s`, `2s`, `4s`, `8s` ... capped at `30s`; restart the mock: `connected:`.
- [ ] Supersede: start a second client (another window): the first logs
      `session ended: websocket closed by kvm (NormalClosure superseded by a new exit)` and
      reconnects after 1 s (the two then take turns; stop one).
- [ ] Idle for 3 minutes, then `host_tests.py --only tcp-echo` PASS. This proves the .NET
      keepalive (30 s) keeps the mock's 20 s ping / 3 misses rule satisfied and that the client
      does not tear down an idle session. (The client cannot see ping/pong frames, so it does not
      apply the 75 s no-frame rule; a dead peer surfaces as a faulted ReceiveAsync or a
      non-Open socket state.)
- [ ] Policy: restart the mock without `--allow-private` (so the served client has
      `__ALLOW_PRIVATE__=0`), reconnect, then on the mock host
      `host_tests.py --socks 127.0.0.1:18081 --bind LAN --expect-policy-deny` PASS
      (rep 0x02 comes from the client's own policy mirror since the mock in this mode would
      also deny; to isolate the client, run the mock with `--allow-private` and edit
      `$NexitAllowPrivate = $false` in a saved copy of the script).

## 5. Against the real NanoKVM (after W5)

- [ ] Paste the Windows command from the UI's Exit panel; `connected:`; the panel shows the peer
      with `os: windows`; a browser on the consumer loads a page; `nslookup` on the consumer works.
- [ ] Regenerate the token in the UI: the client logs `rejected (404) ...` on its next attempt.

## Known limitations to observe, not fix here

- Each `SendAsync` toward the kvm is awaited synchronously: a kvm that stops reading the
  WebSocket stalls the loop. Accepted by the spec (one send in flight). Writes toward
  destinations are asynchronous and queued per stream, so a destination that stops
  reading holds at most one stream window of bytes and stalls nothing else; there is no
  write timeout, the bytes sit until the socket errors or the kvm sends RST, exactly as
  in the Python and Perl clients.
- Only one WebSocket send is ever in flight; the kvm's per-stream window bounds memory,
  on both the read and the write side.
- `Write-Host` output only; nothing is written to files. Run it in its own window.

# Exit tunnel: implementation plan

Spec: `docs/superpowers/specs/2026-09-15-exit-tunnel-design.md` (revised after review; D1–D28 and the `nexit/1` section are normative).
Pinned contracts already in the tree: `server/proto/exit.go`, `server/service/exit/{slot,types,policy}.go`. Build against them; change them only by agreement and say so in the report.

## Waves

Four parallel work packages, each in its own git worktree branch off `main`, touching disjoint files, followed by an integration pass, an adversarial review pass, and the live-test checklist.

| Wave | Package | Branch | Owns |
|---|---|---|---|
| 1 | W1 core | `exit/core` | config, manager, render, downstream, ratelimit, commands, service handlers, router, S94exit, S30rndis change, tunnel.EnsureBinary, presentation OnRebind list, bridge gate, archive.go, system_init.cpp, Makefile, package.sh, .gitignore, CI |
| 1 | W2 dataplane | `exit/dataplane` | socks.go, mux.go, wsproxy.go, dns.go, probe.go and their tests |
| 1 | W3 clients | `exit/clients` | `clients/client.{sh,ps1,pl,py}`, `clients/testdata/mock_kvm.py`, `clients/README.md` |
| 1 | W4 web | `exit/web` | api client, settings panel components, settings tab, 24 locales |
| 2 | W5 integrate | `main` | merge W1–W4, wire, container build, `go test`, `tsc`, conformance suite (python3 + perl clients against the Go mux), fixes |
| 3 | W6 review | workflow | adversarial find → verify → fix |
| 4 | W7 live doc | `main` | `docs/exit-live-test.md` checklist keyed to the task's TESTING list |

## Contracts between packages

### W2 → W1 (constructors the manager calls)

```go
// socks.go
type FrontDoor struct{ /* … */ }
func NewFrontDoor(slot Slot, policy func() Policy, bytes ByteCounter) *FrontDoor
func (f *FrontDoor) Start() error            // binds slot.SocksAddr(); error if taken
func (f *FrontDoor) Stop()
func (f *FrontDoor) SetNative(b Backend)     // Mode A: nil detaches
func (f *FrontDoor) SetRelay(gate RelayGate) // Mode B: relay to slot.WstunnelReverseAddr() while gate.Connected(); nil detaches
func (f *FrontDoor) Attached() (state proto.ExitTunnelState, peer *proto.ExitPeer, since *time.Time)

// mux.go
type MuxHooks struct {
    OnSession func(s Backend)   // after WELCOME; manager calls FrontDoor.SetNative(s) and records the peer
    OnClose   func(s Backend, reason string)
    Policy    func() Policy
    Bytes     ByteCounter
}
// ServeNative upgrades r (already token-authenticated by the caller) and runs the session until it ends.
// A new session closes the previous one on the same Mux (D12). PinPeer refuses a different RemoteAddr.
type Mux struct{ /* … */ }
func NewMux(slot Slot, hooks MuxHooks) *Mux
func (m *Mux) ServeNative(w http.ResponseWriter, r *http.Request)
func (m *Mux) Current() Backend               // nil when none
func (m *Mux) CloseAll(reason string)
func (m *Mux) SetPinPeer(pin bool)

// wsproxy.go (Mode B)
type WSProxy struct{ /* … */ }   // implements RelayGate
func NewWSProxy(slot Slot, bytes ByteCounter, onPeerChange func(prev, next proto.ExitPeer)) *WSProxy
func (p *WSProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) // caller has authenticated; p forwards only Upgrade: websocket && path ends /events, rewrites /exit/<n>/<rest> -> /exit<n>/<rest>, header hygiene per D13/D31
func (p *WSProxy) CloseAll(reason string)
func (p *WSProxy) SetPinPeer(pin bool)

// dns.go
type Forwarder struct{ /* … */ }
func NewForwarder(slot Slot, upstreams func() []netip.Addr, connected func() bool) *Forwarder
func (f *Forwarder) Bind(addr netip.Addr) error   // rebinds UDP+TCP :53 on addr; error surfaces as downstream.dns=false
func (f *Forwarder) Stop()
func (f *Forwarder) Bound() bool
func (f *Forwarder) Stats() proto.ExitDNSStats

// probe.go
type Prober struct{ /* … */ }
func NewProber(slot Slot, connected func() bool) *Prober
func (p *Prober) Start(); func (p *Prober) Stop()
func (p *Prober) Result() proto.ExitUpstream
```

The forwarder and prober dial the front door at `slot.SocksAddr()` as plain SOCKS5 clients (TCP CONNECT), so they work identically in both modes.

### W1 → W2 (what the manager guarantees)

- Exactly one `FrontDoor`, `Mux`, `WSProxy`, `Forwarder`, `Prober` per enabled slot; all stopped on disable.
- Token and rate-limit checks happen in the handler before `Mux.ServeNative` / `WSProxy.ServeHTTP`; both receive only authenticated requests. Rejections are produced by the handler as gin-identical 404s.
- `ByteCounter` implementation lives in W1 (`counter.go`); W2 tests use a trivial local one.

### W3 ↔ W2 (protocol)

Both implement the `nexit/1` section of the spec literally. Placeholders in the client templates, replaced by W1's `commands.go` on serve: `__SCHEME__` (`ws`/`wss` for the socket, `http`/`https` for fetches), `__HOST__`, `__SLOT__`, `__TOKEN__`, `__FINGERPRINT__` (sha256 hex or empty), `__ALLOW_PRIVATE__` (`0`/`1`). No argv. `client.sh` embeds nothing; it fetches `client.py` or `client.pl` with the Authorization header per D14.

### W4 ↔ W1 (API)

Exactly the routes and JSON in the spec's "HTTP surface", in the `{code,msg,data}` envelope. `GET /api/extensions/exit/slots` is the entry point; the panel renders one `ExitSlot` per entry.

## Wave 1 package briefs

### W1 core

Files: `server/service/exit/{config.go,counter.go,manager.go,render.go,downstream.go,ratelimit.go,commands.go,service.go,install.go}` + tests; `server/router/exit.go`; `server/router/router.go` (register `exitRouter(r, presentationManager)` and the post-attach hook: after `presentationManager.Attach` succeeds call `exit.GetManager().OnAttach(ctx)`); `server/service/extensions/tunnel/wrapper.go` (export `EnsureBinary(name string) (path string, err error)` with `.<name>.seed` sha256 staleness and `.<name>.custom` respect; keep existing callers working); `server/service/presentation/manager.go` (`OnRebind` → append to a list; all fire; test); `server/service/bridge/bridge.go` + `bridge/apply.go` (register via list; `Enable` refuses when `exit.AnyEnabled()`, message names the slot); `server/service/application/archive.go` (require `system/init.d/S94exit`); `support/sg2002/kvm_system/main/lib/system_init/system_init.cpp` (copy S94exit next to S29bridge); `kvmapp/system/init.d/S94exit` (new, D8 converge script, `start|stop|status [slot]`, `status` prints `forward=0|1` `routing=…` `tun=…` `hev=…` `wstunnel=…` `nat=…`, reads `/etc/kvm/exit/<n>.json` with a tiny sh JSON extractor or, simpler, W1 renders `/etc/kvm/exit/<n>/env` with `ENABLED= PENDING= MODE= ALLOW_PRIVATE= MTU= NIC=` lines the script sources — do this, and document it in the spec's Data model); `kvmapp/system/init.d/S30rndis` (gateway options iff `/etc/kvm/exit/gadget.route`; treat an ifname containing `(` or `%` as no NIC); `Makefile` (`exit` target per D27, `package` depends on it, help text); `scripts/package.sh` (`require_file`, `require_riscv64`, `require_fresh … third_party/hev-socks5-tunnel "make exit"`, `require_same_gz`); `.gitignore`; `.github/workflows/package.yml`.

Manager: `GetManager()` singleton built lazily with real deps (`presentation.GetManager()`, a `bridgeGate` reading LKG/uplink/`/sys/class/net/br0`/`/boot/rndis.nodhcpd`, an `exec`-based Runner). `Init()` at router time: install/refresh scripts, `EnsureBinary` for hev (and wstunnel if any slot is in Mode B), load slots (`ConfigDir/*.json`, create `0.json` with a fresh token, `enabled:false` if absent), for each enabled slot start front door + forwarder + prober + mux/proxy and run `S94exit start <n>`, then start the watchdog (every 30 s: `S94exit start <n>` converge for enabled slots, `S94exit status <n>` → downstream vector, re-resolve NIC, rebind forwarder if the gadget address changed). `OnAttach`/rebind subscriber per D25. Enable/disable transactions per D23 exactly, with `pending`. Token: `NewToken()` from the 31-symbol alphabet; regenerate per D11. Commands per D15/D16 (pinned wstunnel v10.7.1 URLs and the sha256 table from the release `checksums.txt`, which is in the spec history: linux_amd64 fa842ed5…, linux_arm64 99f9506d…, darwin_amd64 ac234c60…, darwin_arm64 2c1f427f…, windows_amd64 deb3c8b8…, windows_arm64 1d642b29… — fetch the full hashes from `https://github.com/erebe/wstunnel/releases/download/v10.7.1/checksums.txt` and hardcode them). Fingerprint: sha256 of the leaf certificate the server serves (read from the configured cert file; empty on http). Scheme/host from the request per D15.

Handlers (`service.go`): the admin API in the spec plus the token-gated group: `GET /exit/:slot/native` → mux; `GET /exit/:slot/client.{sh,ps1,pl,py}` → templated from `clients/` via `go:embed` (the files come from W3; embed the directory and tolerate its absence at W1 build time by embedding whatever exists — use `embed.FS` and look up by name at request time); `/exit/:slot/*rest` → WSProxy. Token check: `Authorization: Bearer <token>`, `subtle.ConstantTimeCompare`, rate limiter keyed on `RemoteAddr` (IPv6 /64), any failure → `c.AbortWithStatus(404)` producing exactly gin's default 404 (write a test that compares status, headers and body with an unknown route's response). Logs endpoint redacts `[a-z2-9]{8}` tokens.

Tests: config round trip; token alphabet/length/uniqueness; rate limiter (backoff, bounded map, reset on regenerate); render (hev.yml keys exactly as the spec, restrict yaml parses with a yaml library or string assertions, env file); downstream status parsing; commands for every platform × scheme (snapshot tests); 404 identity; manager enable/disable with a fake Runner, fake NICResolver, fake Rebinder, fake BridgeGate (assert step order and rollback on each failing step); presentation OnRebind list test; bridge gate test. S94exit: a shell test that runs the script with `PATH` pointing at stub `ip`/`iptables`/`sysctl`/`start-stop-daemon`/`flock` recording invocations (see `server/service/presentation/testdata/gen_traces.sh` for the idiom) and asserts idempotence (second `start` issues no `-A`/`add` duplicates) and `stop` symmetry.

### W2 dataplane

Files: `server/service/exit/{socks.go,mux.go,wsproxy.go,dns.go,probe.go,udp.go}` + `_test.go`. Use `github.com/gorilla/websocket` (already a dependency) with a dedicated Upgrader (32 KiB buffers, `SetReadLimit(MaxFrame+64)`, `CheckOrigin` permissive: the exit is not a browser). Implement the `nexit/1` section literally: framing, odd stream ids, additive credit with stream 0 as the connection window, WINDOW at half consumed, lifecycle rules, OPEN timer 8 s, HELLO within 5 s, WS ping 20 s / 3 misses, RST-all on WS loss, UDP relay per the UDP subsection. SOCKS5 front door: no-auth only, CONNECT (atyp 1 and 4; atyp 3 → `0x08`), UDP ASSOCIATE with a real loopback relay bound to hev's first datagram source, policy applied before dialing (`0x02`), `0x03` immediately when no backend/relay, Mode B verbatim byte relay to `slot.WstunnelReverseAddr()` gated by `RelayGate.Connected()`. WSProxy per D13/D19/D31 with connection tracking keyed on `RemoteAddr`, supersede on a `/events` upgrade from a new source, `Connected()` true when ≥1 tracked hijacked connection and a TCP probe of the reverse listener succeeded at least once since. DNS forwarder: UDP+TCP on `addr:53`, each query forwarded over TCP through the front door to the upstreams in order with a 3 s budget, SERVFAIL when `!connected()`, AAAA → empty NOERROR, 1 000-entry positive cache honouring TTL (cap 300 s), counters. Prober: TCP CONNECT to `1.1.1.1:443` through the front door every 30 s while connected.

Tests: a Go fake exit implementing the `nexit/1` client side (this fake is also W5's conformance oracle) exercising: handshake, CONNECT round trip with data both ways, half-close both ways, OPEN_FAIL mapping, credit exhaustion and WINDOW resumption, MaxStreams, UDP associate round trip, supersede, HELLO timeout, ping timeout (use short durations via unexported vars), RST-all on WS loss; front door SOCKS conformance with `golang.org/x/net/proxy` as the client; WSProxy against an `httptest` upstream asserting path rewrite, header stripping, 404 for non-upgrade; forwarder against a fake DNS upstream reached through the front door; no listener on `0.0.0.0` (enumerate `net.Listen` addresses in tests). Race detector clean.

### W3 clients

Files: `server/service/exit/clients/{client.sh,client.ps1,client.pl,client.py,README.md}` and `clients/testdata/mock_kvm.py` (a ~200-line Python nexit/1 *server* side that also acts as a SOCKS5 front door on 127.0.0.1:0 so the Perl and Python clients can be exercised end to end on the host without Go: `curl --socks5 127.0.0.1:<p> http://example.com` through the client under test). Implement the per-language architecture in the spec's "Client architecture" subsection: single file, no dependencies beyond the runtime, reconnect loop with backoff 1,2,4…30 s, WS ping 30 s and 75 s dead-peer rule, masked frames, `Sec-WebSocket-Accept` check, fragmentation reassembly, fingerprint pin (`__FINGERPRINT__` non-empty → verify after handshake, else on `wss` refuse unless fingerprint present — never silently unverified), destination policy mirror (`__ALLOW_PRIVATE__`), non-blocking connects with 6 s budget and the errno → reason table, half-close mapping, UDP per stream, HELLO with hostname/os/flags. Python: 3.8+ stdlib only. Perl: 5.18+ core plus `IO::Socket::SSL` only when `wss`. PowerShell: 5.1, `Add-Type` compiled pinning callback, `Tls12,Tls13` in try/catch, task-polled loop. `client.sh` per D14 (Darwin: python3 only if `xcode-select -p` succeeds, else perl; Linux: python3 if `import ssl,socket,selectors` works, else perl; fetch body into a variable, fail loudly if empty). Test locally: run `mock_kvm.py`, run `client.py` and `client.pl` against it with `__SCHEME__=ws`, then `curl --socks5` through the mock's front door for TCP, and `dig @… +tcp`/a UDP echo for UDP. Record results in `clients/README.md`. PowerShell cannot be run here: write it carefully, and include a `clients/testdata/windows-checklist.md`.

### W4 web

Files: `web/src/api/extensions/exit.ts`; `web/src/pages/desktop/menu/settings/exit/{index.tsx,slot.tsx,status.tsx,token.tsx,commands.tsx,types.ts}`; `web/src/pages/desktop/menu/settings/index.tsx` (tab `exit`, icon lucide `RouteIcon` or `GlobeIcon`, admin-only, before `update`); all 24 files in `web/src/i18n/locales/` (`settings.exit.*`, real translations, and `settings.exit.title` in every one — the sidebar renders a raw key otherwise). Follow the conventions in the tunnels spec's "Page" section (named arrow exports, `.ts`/`.tsx` extensions on `@/` imports, `if (isLoading) return`, `.then/.catch/.finally`, `errMsg` in `text-red-500`, antd v5 dark, Tailwind, `ScrollArea`). Panel per slot: enable `Switch` with a confirm `Popconfirm` that says HID/camera re-enumerate for ~2 s; status card (tunnel state coloured like the tunnel page, NIC up/down with ifname and protocol, peer addr/hostname/os with a "changed from …" notice when `previousPeer` is set, upstream reachable + latency, uptime and last connected); token row (monospace, copy button, regenerate with confirm); mode `Segmented` (native / wstunnel) writing `mode` via `setConfig`; platform `Tabs` (Windows PowerShell / macOS / Linux) each with the command in a read-only `Input.TextArea`, a copy button and the D28 warnings; a `Collapse` "Advanced" with DNS servers, MTU, allowPrivate, pinPeer. Poll status every 3 s while the tab is mounted; re-render commands from `window.location` (scheme/host) by substituting the server-provided `scheme`/`host` if they differ. `pnpm exec tsc --noEmit` and `pnpm lint` must pass; add `web/src/pages/desktop/menu/settings/exit/state.test.ts` mirroring the tunnel page's `state.test.ts` if there is a runner (`vitest`?) configured — check `package.json`.

## Wave 2: W5 integration

Merge the four branches into `main` (no rebase; fix conflicts, which should be none). Wire `router/exit.go` to real constructors. `make app DOCKER_TTY= UID=1000 GID=1000 IMAGE_NAME=nanokvm-builder-local-501-20` must succeed. `cd server && go test ./service/exit/... ./service/presentation/... ./service/bridge/... ./service/extensions/tunnel/...` and `go vet` on those packages on the host (they have no cgo). `cd web && pnpm exec tsc --noEmit && pnpm lint`. Conformance: a Go test (build tag `conformance`) that starts the real Mux + FrontDoor on loopback via `httptest`, runs `python3 clients/client.py` and `perl clients/client.pl` (templated with `ws`, the test host, a token) as subprocesses, and drives TCP echo, half-close, OPEN_FAIL, UDP echo and a 20 MB transfer through `golang.org/x/net/proxy`; skip if the interpreter is missing. `make exit DOCKER_TTY= UID=1000 GID=1000 IMAGE_NAME=nanokvm-builder-local-501-20` must produce `kvmapp/exit/hev-socks5-tunnel.gz` with `e_machine 243`. Commit on `main` in logical commits; do not push.

## Wave 3: W6 review

Workflow: finders by lens (protocol conformance vs spec, security of the token-gated surface, S94exit shell correctness, manager transaction/rollback, web UI correctness and i18n parity, packaging) → adversarial verify (2 of 3 refuters) → fix agents → re-verify.

## Wave 4: W7 live-test checklist

`docs/exit-live-test.md`: numbered steps for every item in the task's TESTING list, with the exact commands on the NanoKVM (`ip rule`, `ip route show table 100`, `iptables -S EXIT0`, `S94exit status 0`, `zcat /proc/config.gz | grep …`, `pgrep -a udhcpd`, `ip addr show usb0`), on the consumer (Windows/macOS/Linux DHCP inspection, `nslookup`, `curl`, a 200 MB download, `tracert`), and on the exit device, plus the expected outcomes and what to capture when they differ. Include the hardware-only questions from the review (dnsmasq on :53, `/dev/net/tun`, `ip_forward` after tailscaled, the gadget netdev timing at boot, RSS at 256 sessions).

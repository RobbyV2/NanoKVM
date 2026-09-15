# Reverse-tunnel exit with downstream NIC bridging

Status: design, being implemented. Nothing has run on hardware yet.
Date: 2026-09-15.

## Roles

| Role | Machine | What it does here |
|---|---|---|
| NanoKVM | the RISC-V device | serves the web UI, runs the tunnel server, owns all downstream routing |
| Exit device | whatever browses to the NanoKVM's UI | has real internet, strictly outbound, holds one WebSocket open to the NanoKVM and originates every egress connection |
| Consumer | the machine the NanoKVM is plugged into over USB | installs nothing, sees an ordinary NIC with a default route and DNS |

The goal is a transparent NIC on the consumer whose internet egress happens on the exit device, with the exit to NanoKVM channel carried only over a WebSocket on the same URL the operator used to reach the UI.

## What already exists and is reused

- **The gadget NIC.** `S03usbdev` already builds a composite gadget with HID, optional NCM or RNDIS, optional mass storage and optional UVC, all in one `configs/c.1`. The presentation manager (`server/service/presentation`) owns which functions are linked, publishes the live netdev name through `Manager.NIC()`, and the protocol through `NetworkProtocol()`. Nothing about the composite changes for this feature.
- **The gadget NIC's addressing.** `S30rndis` puts `10.<a>.<b>.1/24` on the gadget NIC (prefix derived from the chip UID, or `/boot/rndis.ipv4_prefix`) and starts BusyBox `udhcpd` with a `.100-.200` pool and a ten-day lease. Today the lease carries no router and no DNS. This feature adds both, conditionally.
- **The wstunnel binary** at `/etc/kvm/bin/wstunnel`, extracted from the `/kvmapp/tunnels/wstunnel.gz` seed by `server/service/extensions/tunnel`. Mode B runs the same binary as a server.
- **The supervision idiom.** An `/etc/init.d/S9x` script with `start-stop-daemon`, plus a 30 s watchdog in the Go server that re-runs `start` when an enabled service's pid is gone (tunnels design D7).
- **Kernel and userland.** The custom kernel (`LicheeRV-Nano-Build` `nanokvm-custom`, config read back from the shipped `boot.sd`) has `CONFIG_TUN=y`, `NF_NAT`, `NFT_NAT`, `NFT_MASQ`, `IP_NF_NAT`, `IP_NF_TARGET_MASQUERADE`, `NETFILTER_XT_TARGET_TCPMSS`, `NETFILTER_XT_MATCH_CONNTRACK`, and `USB_CONFIGFS_{ECM,NCM,RNDIS,EEM}`. Buildroot ships `iptables` (nft backend), `nftables`, `iproute2`, `dnsmasq`. `CONFIG_TUN` and the NAT stack are already `=y` on `upstream/NanoKVM`, so unlike the bridge (whose `CONFIG_BRIDGE` is a fork addition) this feature does not need the custom kernel. The minimal image variant (`sg2002_licheervnano_sd_minimal`) ships only `iproute2` and legacy `iptables`, no `nft`, no `dnsmasq`; nothing here depends on either. `S98tailscaled stop` zeroes `net.ipv4.ip_forward`, so the slot bring-up re-asserts it and the watchdog checks it. `bridge.allowedBinaries` is a closed allowlist local to the bridge package; the exit package runs its own commands (`ip`, `iptables`, `sysctl`, the two daemons) and does not extend it.

## Decisions

| # | Decision |
|---|---|
| D1 | **One internal contract: a loopback SOCKS5 listener per slot, owned by the Go server.** `hev-socks5-tunnel` points at `127.0.0.1:<socksPort>`. Behind that listener the Go server dispatches to whichever exit backend is active: the native mux (Mode A) or a byte relay into wstunnel's reverse-SOCKS listener (Mode B). Nothing downstream branches on mode. The extra loopback hop in Mode B buys a fast, uniform "no exit connected" failure instead of a connection that hangs until wstunnel's idle timeout. |
| D2 | **tun2socks is `hev-socks5-tunnel`**, forked to `RobbyV2/hev-socks5-tunnel` branch `NanoKVM` at tag `2.17.1`, added as `third_party/hev-socks5-tunnel`. C, no dependencies beyond its vendored lwip and task system, riscv64 supported natively. Cross-built statically: 190,544 bytes. Go alternatives (`xjasonlyu/tun2socks`) are 10 MB+ and a second gvisor netstack on a 256 MB device. |
| D3 | **DNS is a Go forwarder inside NanoKVM-Server**, bound to the gadget NIC address on port 53 (UDP and TCP), forwarding every query over TCP through the slot's SOCKS listener to pinned public resolvers (`1.1.1.1`, `8.8.8.8`, configurable). It never consults `/etc/resolv.conf` and answers SERVFAIL when no exit is connected. This makes DNS independent of SOCKS UDP support and gives the "raw IP works but names do not" failure no place to hide. dnsmasq is not used: it cannot be told to use TCP upstream, and UDP-over-SOCKS is the least tested path in both exits. |
| D4 | **DHCP stays with `udhcpd`** as started by `S30rndis`. When a slot is enabled the Go server writes `/etc/kvm/exit/<slot>/dhcp-options` containing `opt router <gw>` and `opt dns <gw>`; `S30rndis` appends every `/etc/kvm/exit/*/dhcp-options` to the config it generates. Disabling removes the file. No second DHCP server, no change to the pool or lease. |
| D5 | **Enable and disable re-lease the consumer by a gadget pull-up cycle.** A ten-day lease means a consumer would otherwise not see the new router and DNS options for days. The enable transaction asks the presentation manager to detach and re-attach the gadget once, which the consumer sees as the NIC (and keyboard, mouse, camera) briefly re-enumerating, then re-DHCPs. This happens only on the explicit enable or disable, never on exit connect or disconnect. |
| D6 | **Routing is policy-based, keyed on the gadget NIC.** `ip rule add iif <gadget> lookup <100+slot>`; table `100+slot` holds `default dev exit<slot>`. The NanoKVM's own traffic, including the UI and the management address, never touches the tun. Locally addressed packets from the consumer hit the `local` table first, so intranet reachability of the NanoKVM survives with or without an exit. `iptables -t nat -A POSTROUTING -o exit<slot> -j MASQUERADE`, `FORWARD` accepts between the gadget NIC and the tun, and `TCPMSS --clamp-mss-to-pmtu` on `FORWARD -o exit<slot>`. `net.ipv4.ip_forward=1`. |
| D7 | **tun MTU 1280**, both on `hev-socks5-tunnel`'s `tunnel.mtu` and on the tun itself. IP in SOCKS in WS in TLS otherwise black-holes large packets. The MSS clamp in D6 makes consumer TCP honour it. |
| D8 | **The downstream half is unconditionally up while enabled.** `S94exit` (before `S95nanokvm`) brings up, per enabled slot: sysctl, the tun via `hev-socks5-tunnel`, the routing table and rule, the iptables rules. It does not depend on the server. The server adds the DNS forwarder, the SOCKS front door, the WS endpoint and the watchdog at S95. Exit connect and disconnect touch none of it. Only an explicit disable tears it down. |
| D9 | **Everything is scoped by a slot id.** Slot `n` has: config `/etc/kvm/exit/<n>.json`, tun `exit<n>` at `198.18.<n>.1/30`, routing table `100+n`, SOCKS front door `127.0.0.1:10800+n`, wstunnel server `127.0.0.1:10810+n`, wstunnel reverse listener `127.0.0.1:10820+n`, WS path `/exit/<n>/<token>/...`, and a UI panel keyed by `n`. One slot, `0`, exists today; the slot list is `/etc/kvm/exit/*.json`. The one gadget NIC is bound to slot 0 through the config's `nic: "gadget"`; a second NIC would be a second value of that field. |
| D10 | **Token: 8 characters from `abcdefghjkmnpqrstuvwxyz23456789`** (no `0 o 1 l i`), lowercase, generated from `crypto/rand`, 32^8 ≈ 1.1×10¹² values. Always visible in the panel. Validated on every WS upgrade and script fetch with a constant-time compare. Rejections return a plain `404` identical to an unknown route. Rate limit: 5 failures per source IP within 10 minutes locks that IP out for 15 minutes; 30 failures per minute from anywhere pauses all validation for 60 seconds. At the per-IP limit a single attacker gets 20 guesses an hour; the global limit caps a distributed attacker at 30 a minute. |
| D11 | **Regenerating the token** writes the new token, closes every active exit session on that slot (Mode A mux, Mode B tracked upgraded connections), and restarts the slot's wstunnel server so its own `--restrict-http-upgrade-path-prefix` follows. |
| D12 | **One exit per slot; a new valid connection supersedes the old.** Mode A: a new mux handshake closes the previous session. Mode B: wstunnel's client opens one WS per stream, all sharing one client; the Go proxy tracks upgraded connections by source address, and a `/events` upgrade from a new source closes every tracked connection from other sources. Staleness is caught by WS pings: the Go mux pings every 20 s and drops a session that misses three; wstunnel clients ping every 30 s by default and the server drops dead ones. |
| D13 | **Mode B runs wstunnel server behind the Go server**, on `127.0.0.1:10810+n`, `--restrict-config` allowing only `ReverseTunnel` of protocol `Socks5` on port `10820+n` bound to `127.0.0.1`, matched by path prefix `^<token>$`. The Go route `/exit/:slot/:token/*rest` validates slot and token, rewrites the path to `/<token>/<rest>` and reverse-proxies with `httputil.ReverseProxy`, which handles the 101 upgrade. wstunnel's server-side `extract_path_prefix` takes the first path segment as the prefix, so the client is told `-P exit/<n>/<token>` and its `/exit/<n>/<token>/events` reaches the Go route unchanged. |
| D14 | **Mode A is a protocol this repo owns, `nexit/1`**, terminated in Go and implemented three times on the exit side: PowerShell 5.1 (Windows, `System.Net.WebSockets.ClientWebSocket` and `TcpClient`), Perl 5 (macOS, `IO::Socket::SSL` ships with the system perl, verified 2.068 on macOS 26), Python 3 (Linux, stdlib `ssl`, `socket`, `select`). The one-liners fetch the script from the NanoKVM itself at `/exit/<n>/<token>/client.{ps1,pl,py}` and run it in-process. The NanoKVM is the only thing fetched from and nothing is written to disk on the exit device. The `sh` one-liners try `python3` first and fall back to `perl`, so a macOS with the Xcode tools and a Linux with only perl both work. |
| D15 | **Commands are templated from the request**, on the server, from `Host` and TLS state (or `X-Forwarded-Proto` when the UI is reached through a proxy), and re-rendered in the UI from `window.location` so they follow however the operator reached the page. |
| D16 | **wstunnel v10.7.1 is pinned in Mode B commands**, URL and SHA-256 per asset, from the upstream release `checksums.txt`. Windows and macOS commands verify the hash before running; Linux does too. The platform command detects `amd64`/`arm64` at paste time; the hash table covers both. |
| D17 | **Bridge and exit are mutually exclusive on the gadget NIC.** An enslaved NIC has no address and the DHCP server that would hand out the exit's gateway is suppressed by `/boot/rndis.nodhcpd`. Enabling exit while the bridge's last-known-good says enabled fails with a message, and vice versa. |
| D18 | **Enabled state is the `enabled` field of the slot config**, not the presence of an init script, because one script serves every slot. `S94exit` is always installed; it does nothing for disabled slots. |
| D19 | Status is polled by the UI every 3 s like the tunnel and tailscale pages. The server keeps per-slot state in memory: tunnel state, exit peer, connect time, last connected time, bytes, probe result. `lastConnectedAt` is also written to `/etc/kvm/exit/<n>.state.json` so it survives a server restart. |
| D20 | **Probe.** While an exit is connected, every 30 s the server dials `1.1.1.1:443` through the slot's SOCKS front door with a 5 s timeout and records reachability and latency. The DNS forwarder's success rate is a second signal. Nothing is probed when no exit is connected. |

## Data model

`/etc/kvm/exit/<n>.json`, mode 0600, written atomically:

```json
{
  "slot": "0",
  "enabled": true,
  "mode": "native",
  "token": "k7m2p9vx",
  "tokenCreatedAt": "2026-09-15T20:00:00Z",
  "nic": "gadget",
  "dns": ["1.1.1.1", "8.8.8.8"],
  "mtu": 1280
}
```

Derived, never stored: ports, tun name, table id, tun address, paths.

`/etc/kvm/exit/<n>/dhcp-options` (D4), `/etc/kvm/exit/<n>/hev.yml`, `/etc/kvm/exit/<n>/wstunnel-restrict.yml`, `/etc/kvm/exit/<n>.state.json`. Logs at `/tmp/exit<n>-hev.log` and `/tmp/exit<n>-wstunnel.log`. Pidfiles `/var/run/exit<n>-hev.pid`, `/var/run/exit<n>-wstunnel.pid`.

## Processes per slot

| Process | Binary | Started by | Bound to |
|---|---|---|---|
| `hev-socks5-tunnel` | `/etc/kvm/bin/hev-socks5-tunnel`, seed `/kvmapp/exit/hev-socks5-tunnel.gz` | `S94exit`, watchdog | creates `exit<n>`, talks to `127.0.0.1:10800+n` |
| `wstunnel server` (Mode B only) | `/etc/kvm/bin/wstunnel` | `S94exit`, watchdog | `127.0.0.1:10810+n`; reverse listener `127.0.0.1:10820+n` |
| SOCKS front door | in NanoKVM-Server | server start | `127.0.0.1:10800+n` |
| DNS forwarder | in NanoKVM-Server | server start, re-bound when the gadget NIC address changes | `<gadget ip>:53` UDP and TCP |
| WS endpoint | in NanoKVM-Server | route | `/exit/<n>/<token>/...` |

The SOCKS front door and the DNS forwarder are unconditional while the slot is enabled. The front door answers SOCKS5 immediately and fails the CONNECT with `0x03 network unreachable` in under a millisecond when no exit is attached, so a consumer with no exit sees connection refused, not a hang.

## The `nexit/1` protocol (Mode A)

One WebSocket, binary frames, every frame `type:u8, stream:u32be, payload`. Stream ids are allocated by the NanoKVM, odd numbers, so a future exit-initiated stream cannot collide.

| Type | Name | Direction | Payload |
|---|---|---|---|
| 0x01 | HELLO | exit → kvm | `version:u8=1, flags:u8, hostname:len-prefixed utf8, os:len-prefixed utf8` |
| 0x02 | WELCOME | kvm → exit | `version:u8=1, window:u32be` initial per-stream credit |
| 0x10 | OPEN | kvm → exit | `proto:u8 (1=tcp,2=udp), atyp:u8 (1=ipv4,3=domain,4=ipv6), addr, port:u16be` |
| 0x11 | OPENED | exit → kvm | `atyp, bound addr, port` |
| 0x12 | OPEN_FAIL | exit → kvm | `reason:u8` mapped to SOCKS reply codes |
| 0x20 | DATA | both | bytes (TCP) or `atyp, addr, port, datagram` (UDP) |
| 0x21 | EOF | both | none; half close |
| 0x22 | RST | both | none; abort |
| 0x30 | WINDOW | both | `credit:u32be` bytes the sender may now send on this stream |
| 0x40 | PING / 0x41 PONG | both | opaque 8 bytes |

Flow control is credit based: each side starts with `window` bytes per stream (256 KiB), DATA consumes it, and the receiver returns WINDOW once it has passed half the window on to its socket. Frames are at most 16 KiB of payload. The Go side never buffers more than one window per stream, which bounds memory on the device.

The HELLO is the first frame and must arrive within 5 s of the upgrade. WELCOME closes any previous session on the slot (D12). WS-level ping every 20 s from the NanoKVM; three misses close the session.

## HTTP surface

Token-gated, no JWT, outside `/api`:

```
GET  /exit/:slot/:token/native           nexit/1 WebSocket upgrade
GET  /exit/:slot/:token/client.ps1       PowerShell client
GET  /exit/:slot/:token/client.pl        Perl client
GET  /exit/:slot/:token/client.py        Python client
ANY  /exit/:slot/:token/*rest            Mode B: reverse-proxied to wstunnel server as /:token/*rest
```

Admin API, JWT and `RequireRole(admin)`, under `/api/extensions/exit`:

```
GET    /slots                          list slots with status
GET    /:slot/status
GET    /:slot/config
POST   /:slot/config                   { mode, dns?, mtu? }
POST   /:slot/enable
POST   /:slot/disable
POST   /:slot/token/regenerate
POST   /:slot/disconnect               drop the active exit
GET    /:slot/commands                 templated commands for both modes and three platforms
GET    /:slot/logs
```

Status response:

```json
{
  "slot": "0", "enabled": true, "mode": "native",
  "tunnel": "connected",
  "peer": { "addr": "203.0.113.7", "hostname": "robs-laptop", "os": "darwin", "transport": "native" },
  "connectedAt": "…", "lastConnectedAt": "…", "uptimeSeconds": 812,
  "nic": { "ifname": "usb0", "up": true, "protocol": "ncm", "address": "10.12.34.1/24", "leases": 1 },
  "downstream": { "tun": true, "hev": true, "wstunnel": true, "dns": true, "nat": true },
  "upstream": { "reachable": true, "latencyMs": 41, "checkedAt": "…" },
  "bytes": { "up": 1234, "down": 5678 },
  "message": ""
}
```

`tunnel` is one of `disconnected`, `connecting`, `connected`. `connecting` means an upgrade has been accepted but the HELLO (Mode A) or the reverse listener (Mode B) has not appeared yet.

## Enable transaction

1. Refuse if the bridge is enabled (D17) or the presentation profile links no network function (message tells the operator to turn on Virtual Network under Settings, Device).
2. Extract seeds if missing: `hev-socks5-tunnel`, `wstunnel`.
3. Write `hev.yml`, `wstunnel-restrict.yml`, `dhcp-options`, then the config with `enabled: true`.
4. `S94exit start <slot>` brings up sysctl, tun, routes, rules, iptables, processes. Idempotent.
5. Start the SOCKS front door and DNS forwarder in-process.
6. `S30rndis restart` regenerates the udhcpd config with the options and restarts udhcpd (leases persist in its lease file).
7. Presentation manager gadget pull-up cycle (D5).

Disable reverses 3–7 and additionally removes routes, rules and iptables entries for the slot, kills both daemons, and deletes the tun. The consumer's NIC comes back after the cycle with a lease that carries no router and no DNS, exactly the stock state.

## Boot

`S30rndis` (address, udhcpd with options) → `S94exit` (tun, routing, NAT, daemons for each enabled slot) → `S95nanokvm` (server: front door, DNS, WS endpoint, watchdog). The consumer sees the NIC as soon as the gadget binds and gets a lease with the gateway immediately; names fail with SERVFAIL until the server is up, then until an exit connects; then everything flows.

## Files

Backend:

| Path | Action |
|---|---|
| `server/proto/exit.go` | new: types, states, requests, responses |
| `server/service/exit/slot.go` | new: slot id, derived names and ports |
| `server/service/exit/config.go` | new: atomic load and save, token generation |
| `server/service/exit/manager.go` | new: per-slot runtime state, enable and disable transactions, watchdog |
| `server/service/exit/socks.go` | new: SOCKS5 front door (CONNECT and UDP ASSOCIATE), backend dispatch |
| `server/service/exit/mux.go` | new: `nexit/1` server side |
| `server/service/exit/wstunnel.go` | new: restrict yaml, args, reverse proxy with connection tracking |
| `server/service/exit/dns.go` | new: forwarder over SOCKS TCP |
| `server/service/exit/probe.go` | new: reachability probe |
| `server/service/exit/ratelimit.go` | new: token attempt limiter |
| `server/service/exit/commands.go` | new: command templating, pinned wstunnel table |
| `server/service/exit/clients/*.{ps1,pl,py}` | new: embedded native clients (`go:embed`) |
| `server/service/exit/service.go` | new: gin handlers |
| `server/service/exit/downstream.go` | new: hev.yml rendering, dhcp-options, calls into S94exit/S30rndis |
| `server/router/exit.go` | new: both route groups |
| `server/router/router.go` | modify: register |
| `server/service/extensions/tunnel/wrapper.go` | modify: export `EnsureBinary(name)` for reuse |

Device and build:

| Path | Action |
|---|---|
| `kvmapp/system/init.d/S94exit` | new |
| `kvmapp/system/init.d/S30rndis` | modify: append `/etc/kvm/exit/*/dhcp-options` |
| `.gitmodules`, `third_party/hev-socks5-tunnel` | new submodule |
| `Makefile` | `exit` target cross-building hev-socks5-tunnel into `kvmapp/exit/hev-socks5-tunnel.gz`; `package` depends on it |
| `scripts/package.sh` | `require_riscv64` on the new seed |
| `.gitignore` | `kvmapp/exit/*.gz` |
| `.github/workflows/package.yml` | `make exit` |

Frontend:

| Path | Action |
|---|---|
| `web/src/api/extensions/exit.ts` | new |
| `web/src/pages/desktop/menu/settings/exit/index.tsx` | new: renders one `ExitSlot` per slot from `/slots` |
| `web/src/pages/desktop/menu/settings/exit/slot.tsx` | new: the repeatable unit |
| `web/src/pages/desktop/menu/settings/exit/status.tsx` | new |
| `web/src/pages/desktop/menu/settings/exit/token.tsx` | new |
| `web/src/pages/desktop/menu/settings/exit/commands.tsx` | new: mode tabs, platform sub-tabs, copy buttons |
| `web/src/pages/desktop/menu/settings/exit/types.ts` | new |
| `web/src/pages/desktop/menu/settings/index.tsx` | modify: one tab `exit` |
| `web/src/i18n/locales/*.ts` | modify: `settings.exit.*` in all 24 |

## Testing

Unit, in Go: config round trip, token alphabet and length, rate limiter windows, SOCKS5 handshake parsing, `nexit/1` framing and flow control against an in-process fake exit, DNS forwarder against a fake upstream over a fake SOCKS, command templating for every platform and scheme, path rewrite for the wstunnel proxy, restrict yaml rendering, hev.yml rendering.

Integration, on the host: the three native clients against the Go mux (run the Python and Perl clients locally against an `httptest` server; PowerShell is exercised on Windows in live testing), the Mode B proxy against a real `wstunnel` binary (host build) with the client in reverse-SOCKS mode, end to end `curl --socks5` through the front door.

Live, on hardware: everything in the task's TESTING list, driven from the checklist in `docs/exit-live-test.md` written alongside the implementation.

## Risks

1. `hev-socks5-tunnel` has never run on this kernel or CPU. First check: `hev-socks5-tunnel <conf>` brings up `exit0`, `ip addr` shows `198.18.0.1`, and `curl --interface exit0` fails fast with no exit.
2. Gadget pull-up cycle on enable drops HID for about two seconds. Deliberate, documented in the UI.
3. SOCKS UDP through wstunnel's reverse SOCKS is the least exercised path. DNS does not depend on it (D3); QUIC falls back to TCP.
4. RAM. hev-socks5-tunnel is small; wstunnel server is the same binary already measured. The Go side bounds buffering per stream to one window.
5. The consumer keeps our default route even when it has another uplink. That is the point of the feature and also why the options are only handed out while enabled.

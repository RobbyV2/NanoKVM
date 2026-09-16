# Reverse-tunnel exit with downstream NIC bridging

Status: design revised after a five-lens adversarial review (networking, USB gadget, security, exit clients, resilience); implemented and integrated (plan Waves 1 and 2). Where the implementation deviates from a decision, the deviation is recorded in that decision's text. Nothing has run on hardware yet.
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
| D1 | **One internal contract: a loopback SOCKS5 listener per slot, owned by the Go server.** `hev-socks5-tunnel` points at `127.0.0.1:<socksPort>`. Behind that listener the Go server dispatches to whichever exit backend is active: the native mux (Mode A) or a byte relay into wstunnel's reverse-SOCKS listener (Mode B). Nothing downstream branches on mode. The front door is also the single chokepoint for the destination policy (D21) and owns the SOCKS UDP ASSOCIATE relay (see the `nexit/1` section). In Mode B the SOCKS bytes are relayed verbatim so wstunnel's own UDP server address reaches hev; hev's datagrams then go straight to wstunnel's UDP server, so in Mode B the front door cannot apply D21 per datagram (only the `EXIT<n>` chain and wstunnel's restrictions apply there), while Mode A applies it to every datagram. The extra loopback hop buys a fast, uniform "no exit connected" failure (`0x03` in under a millisecond) instead of a hang. The front door refuses domain CONNECTs with `0x08`; hev only ever sends literal IPs. |
| D2 | **tun2socks is `hev-socks5-tunnel`**, forked to `RobbyV2/hev-socks5-tunnel` branch `NanoKVM` at tag `2.17.1`, added as `third_party/hev-socks5-tunnel`. C, no dependencies beyond its vendored lwip and task system, riscv64 supported natively. Cross-built statically: 190,544 bytes. It attaches to a tun that S94exit has already created (D6); its own MTU, address and UP ioctls are redundant no-ops. The fork may set the lwip `netif->mtu` from `tunnel.mtu` so lwip advertises a correct MSS; the clamp in D7 does not depend on it. Go alternatives are 10 MB+ and a second netstack on a 256 MB device. |
| D3 | **DNS is a Go forwarder inside NanoKVM-Server**, bound to the gadget NIC address on port 53 (UDP and TCP), forwarding every query over TCP through the slot's SOCKS listener to pinned public resolvers (`1.1.1.1`, `8.8.8.8`, configurable). It never consults `/etc/resolv.conf`, answers SERVFAIL when no exit is connected, and answers AAAA with an empty NOERROR while the slot carries no IPv6 (D22). All plaintext DNS from the consumer is DNAT'd to it (`EXIT<n>_NAT` PREROUTING, udp and tcp 53, `! -d <gw>`), so a consumer that ignores the lease's resolver still terminates here; the count is `dns.redirected` in status, read by `S94exit status` from the DNAT rules' packet counters. Because every converge flushes and repopulates the chain (which zeroes the counters), `S94exit start` adds the hits so far to `/var/run/exit<n>.dns_redirected` before the flush and `status` prints carried + live: a total since enable (or since boot, the file is on tmpfs; `stop` removes it). Packets that hit between the read and the flush are lost, which is the "best effort" this counter was always meant to be. The bind happens only after the gadget address exists (or with `IP_FREEBIND`) and a bind failure (dnsmasq from `S80dnsmasq` holding `0.0.0.0:53` is the known case; if `/etc/dnsmasq.conf` exists, a drop-in `except-interface=<gadget>` + `bind-dynamic` is written and dnsmasq restarted) fails the enable transaction and shows as `downstream.dns=false`. dnsmasq is not used as the forwarder: it cannot be told to use TCP upstream. |
| D4 | **DHCP stays with `udhcpd`** as started by `S30rndis`, and `S30rndis` owns the gateway value. When the downstream half of a slot with `nic: "gadget"` has been verified, the server writes the marker `/etc/kvm/exit/gadget.route` (content: the slot id); `gen_udhcpd_conf` emits `opt router ${ipv4_prefix}.1` and `opt dns ${ipv4_prefix}.1` iff that marker exists. Disable removes the marker first. The server never recomputes the prefix; it reads the gateway from the live NIC for the DNS bind and for status. No second DHCP server, no change to the pool or lease. |
| D5 | **Enable and disable re-lease the consumer through `presentation.Manager.Rebind(ctx)`.** A ten-day lease means a consumer would otherwise not see the new router and DNS options for days. The cycle re-enumerates the whole composite (keyboard, mouse, camera, virtual disk, NIC) and the consumer re-DHCPs. A refusal (`ErrTransient` during a functionfs session, `ErrUDCLoaned` during passthrough, media busy) is not an enable failure: the transaction completes and status carries "re-plug the USB cable to pick up the new gateway". The design called for polling the UDC state for `configured` after the cycle the way `confirmEnumeration` does; presentation exposes no such method to this package, so instead Enable runs one more converge after the rebind (re-resolve the NIC, `S94exit start`, `S30rndis start` if unaddressed, forwarder rebind), and a refused rebind is reported in `message`. A small exported presentation helper would close this. This happens only on the explicit enable or disable, never on exit connect or disconnect. |
| D6 | **Routing is policy-based, keyed on the gadget NIC, and fails closed.** S94exit creates and owns a persistent tun `exit<n>` (`ip tuntap add … mode tun`, `198.18.<n>.1/32`, MTU per D7, up) so routes never disappear with hev. Table `100+n` holds `default dev exit<n>` and `unreachable default metric 4294967295`. Rules: `pref 1000+2n iif <gadget> lookup 100+n` and `pref 1001+2n iif <gadget> unreachable` (both below Tailscale's 5210+; two prefs per slot so slots never collide). The NanoKVM's own traffic never touches the tun; consumer packets to the NanoKVM's addresses hit `local` first. Per-slot chains, flushed and repopulated on every `S94exit start`: `EXIT<n>` jumped from FORWARD with, in order, REJECT for non-global destinations unless `allowPrivate` (D21), both MSS clamps (D7; they precede the ACCEPTs because ACCEPT ends filter traversal and TCPMSS does not), `-i <gadget> -o exit<n> -j ACCEPT`, `-i exit<n> -o <gadget> -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT`, then `-i <gadget> -j REJECT --reject-with icmp-admin-prohibited` and `-o <gadget> ! -i exit<n> -j REJECT`; `EXIT<n>_NAT` from PREROUTING with the DNS DNAT (D3) and `EXIT<n>_MASQ` from POSTROUTING with `-s <gadget subnet> -o exit<n> -j MASQUERADE` (two chains because iptables refuses a DNAT in a chain reachable from POSTROUTING); `ip6tables -A FORWARD -i <gadget> -j REJECT --reject-with icmp6-adm-prohibited`. Sysctls: `net.ipv4.ip_forward=1`, `net.ipv4.conf.<gadget>.route_localnet=0`, `net.ipv4.conf.exit<n>.rp_filter=2`, `net.ipv6.conf.<gadget>.disable_ipv6=1`. No FORWARD policy is changed. With hev down the consumer sees drops (persistent tun, no reader) or ICMP unreachable (fences), never a packet on the LAN. |
| D7 | **tun MTU 1280**, on the tun (set by S94exit) and in `tunnel.mtu`. IP in SOCKS in WS in TLS otherwise black-holes large packets. TCP MSS is clamped in both directions to a fixed `mtu-40` (`-p tcp --tcp-flags SYN,RST SYN -j TCPMSS --set-mss 1240`, `-i <gadget> -o exit<n>` and `-i exit<n> -o <gadget>`): lwip's compiled `TCP_MSS` is 8191 and hev never sets the netif MTU, so without the tun→gadget clamp the SYN-ACK advertises 8191. Optionally `opt mtu 1280` in the lease. |
| D8 | **The downstream half is unconditionally up while enabled, and `S94exit start <slot>` is a converge, not a start.** It is idempotent and lock-serialised (`flock /var/run/exit<n>.lock`): sysctls; tun if absent; `ip route replace`; `ip rule del pref` loop then add; chains `-N` (EEXIST ignored), `-F`, repopulate, jump by `-C || -I`; hev (and wstunnel in Mode B) via `start-stop-daemon -S -bmq -p /var/run/exit<n>-<daemon>.pid -x /bin/sh -- -c 'exec >>"$1" 2>&1; shift; exec "$@"' sh /tmp/exit<n>-<daemon>.log <binary> <args…> 9>&-`: BusyBox's `-b` gives the daemon `/dev/null` stdio, so the log redirection happens in a shell that becomes the daemon by exec (the pidfile still names the daemon, and wstunnel's stderr-only log reaches `/tmp/exit<n>-wstunnel.log`), and the slot lock's fd 9 is closed for the daemon alone because an inherited open file description would hold the flock for the daemon's whole lifetime and block every later converge; "already running" is success, `misc.pid-file` is never set. It resolves the gadget NIC with the same `gadget_nic()` loop as S30rndis, treats `(unnamed net_device)` as absent, and skips the NIC-keyed rules when no netdev is registered yet (at boot the netdev appears only when the server binds the UDC at S95, or at S03usbdev's 30 s fallback). The server re-runs `S94exit start <slot>` and `S30rndis start` right after `Attach` and on every rebind (D25). `S94exit status <slot>` exits non-zero if ip_forward, either rule, the fence route, the chain jump, tun carrier, or a daemon pid is missing; the Go watchdog runs `start <slot>` every 30 s for every enabled slot regardless of pid and publishes `status` as the `downstream` block. The watchdog and the post-attach/rebind converge yield to a running transaction (a `TryLock` on the transaction mutex) rather than queue behind it: `start <slot>` does not check `ENABLED`, so a converge that had passed the enabled check and then ran after Disable's `stop` would bring the slot back on a config that says disabled; the transaction leaves the slot converged itself. The same converge starts the front door, forwarder and prober of an enabled slot that has none (the SOCKS port was held by a stale process at boot, or a mode switch's restart failed), so an enabled slot never stays with hev dialling a closed port until an operator toggles it; the failure stays in `message` until a retry succeeds. If the hev or wstunnel binary is missing, S94exit extracts it from `/kvmapp/exit/*.gz` itself. No-argument `start`/`stop` iterate slots with `enabled: true` and no `pending`. `stop <slot>` reverses everything by identity (pidfile kill, `-D` jump, `-F`/`-X`, `ip rule del pref`, `ip route flush table`, `ip link del`). Exit connect and disconnect touch none of it. |
| D9 | **Everything is scoped by a slot id.** Slot `n` has: config `/etc/kvm/exit/<n>.json`, tun `exit<n>` at `198.18.<n>.1/32`, routing table `100+n`, rule prefs `1000+2n`/`1001+2n`, chains `EXIT<n>`/`EXIT<n>_NAT`/`EXIT<n>_MASQ`, SOCKS front door `127.0.0.1:10800+n`, wstunnel server `127.0.0.1:10810+n`, wstunnel reverse listener `127.0.0.1:10820+n`, lock `/var/run/exit<n>.lock`, HTTP routes under `/exit/<n>/` (token in the `Authorization` header, D10), and a UI panel keyed by `n`. One slot, `0`, exists today. The one gadget NIC is bound to slot 0 through `nic: "gadget"`; the `gadget.route` marker (D4) is keyed by that role, not by slot. |
| D10 | **Token: 8 characters from `abcdefghjkmnpqrstuvwxyz23456789`** (31 symbols; no `0 o 1 l i`), lowercase, from `crypto/rand`, 31^8 ≈ 8.5×10¹¹ values. Carried as `Authorization: Bearer <token>` on every WS upgrade, script fetch and Mode B request, never in the path, so it does not reach wstunnel's log, proxy access logs or `/:slot/logs`. Constant-time compare, and only against a token this package issued: a slot file whose token is empty or otherwise not `ValidToken` (hand-edited, restored, missing from the JSON) is repaired at load with a fresh token and forced `enabled: false`, and the gate itself refuses to match anything against an invalid stored token, so an absent header can never equal an empty token. Rejections (unknown slot, wrong token, locked source, even a correct token from a locked source) return a 404 byte-identical to gin's default. Rate limit: per source only, keyed on `RemoteAddr` (IPv6 by /64), exponential backoff, always on, bounded map as in `auth/brute_force.go`; no global pause. Only token attempts count (unknown slot, wrong token); a correct token on a slot that is disabled or mid-transaction is refused with the same 404 but not counted, so an exit's reconnect loop during a disable/enable cannot lock it out of the slot. `X-Forwarded-For` would be consulted only behind a trusted-proxy CIDR; no such setting exists in the config, so it is never consulted and the limiter keys on `RemoteAddr` alone. Every rejection, whoever produces it (the handler, the mux's pin-peer refusal, the proxy's non-upgrade and pin-peer refusals), goes through one `writeNotFound` helper, and a test compares each path against gin's unknown-route 404 over a real listener. The token is still in the one-liner and therefore in the operator's shell history; the UI says so. |
| D11 | **Regenerating the token** writes the new token, atomically rewrites `wstunnel-restrict.yml` (wstunnel hot-reloads it; no restart), closes every active exit session on that slot, and clears the rate limiter entry for the peer that was connected. The UI tells the operator to stop the old client first. |
| D12 | **One exit per slot; a new valid connection supersedes the old.** Mode A: a new mux handshake closes the previous session. Mode B: the Go proxy tracks upgraded connections by `RemoteAddr`, and a `/events` upgrade from a new source closes every tracked connection from other sources. Every supersede is logged at warn level with both addresses and recorded as `previousPeer`/`peerChangedAt` in status; the panel shows a notice when the peer changes. Optional per-slot "pin to first peer" refuses supersede from a different source until cleared. Staleness is caught by WS pings (mux: 20 s, three misses; wstunnel: its own 30 s). Status `connecting` comes from `Mux.Connecting()` in Mode A (upgrade accepted, HELLO pending) and from the front door in Mode B (peer tracked, reverse listener unproven). Bytes are counted once per mode: the mux counts DATA payload in Mode A, the front door counts relayed payload in Mode B, and the WS proxy counts nothing. |
| D13 | **Mode B runs wstunnel server behind the Go server**, on `127.0.0.1:10810+n`, with `--restrict-config` alone (`--restrict-http-upgrade-path-prefix` and `--restrict-to` conflict with it). The yaml has exactly one restriction: `match: [!PathPrefix '^exit<n>$', !Authorization '^Bearer <token>$']`, `allow: [!ReverseTunnel {protocol: [Socks5], port: ['10820+n'], cidr: ['127.0.0.1/32']}]`, no `!Tunnel` entry. The Go route `/exit/:slot/*rest` validates the header, forwards only `Upgrade: websocket` requests whose rest ends in `/events`, rewrites to `/exit<n>/<rest>`, forwards `Authorization`, strips `Cookie` and incoming `X-Forwarded-*`/`Forwarded`, calls `SetXForwarded()`, and proxies with `httputil.ReverseProxy`. The client is told `-P exit/<n> -H "Authorization: Bearer <token>"`. Tests: the rendered yaml parses in the real binary; a `-L` forward tunnel and `-R tcp://0.0.0.0:22` are refused. |
| D14 | **Mode A is a protocol this repo owns, `nexit/1`**, terminated in Go and implemented three times on the exit side: PowerShell 5.1, Perl 5 (`IO::Socket::IP` core, `IO::Socket::SSL` only for `wss`), Python 3 (stdlib). Scripts are templated on the server with scheme, host, slot, token and the device certificate's SHA-256, and each client pins that fingerprint after the TLS handshake. Unix: `curl -fsSL[k] -H 'Authorization: Bearer <t>' <base>/client.sh | sh`; `client.sh` fetches the chosen client into a variable, fails loudly on an empty body, picks python3 on Darwin only if `xcode-select -p` succeeds (Apple's shim pops a dialog otherwise) and on Linux only if `ssl,socket,selectors` import, else perl, and runs it with `-c`/`-e`. Windows: `irm -Headers @{Authorization='Bearer <t>'} <base>/client.ps1 | iex`, prefixed on https by `[Net.ServicePointManager]::ServerCertificateValidationCallback={$true};` for the fetch; client.ps1 then installs a compiled (`Add-Type`) pinning callback, which writes a helper under `%TEMP%` and needs Full Language Mode and Windows 8+. Scripts are served `text/plain; charset=utf-8`. One scheme per script, no fallback; the reconnect loop lives in the fetched script. Native clients also refuse the always-deny destinations of D21. The PowerShell client cannot see ping/pong frames through `ClientWebSocket`, so it does not apply the 75 s no-frame rule literally: it sends keepalives through `Options.KeepAliveInterval` (30 s), relies on the kvm's 20 s ping / three misses, and treats a faulted `ReceiveAsync` as the session end. It has been executed under PowerShell 7 (`pwsh` 7.6.6) on macOS against `clients/testdata/mock_kvm.py`, where every `host_tests.py` test passes, but never under Windows PowerShell 5.1 or .NET Framework (no Windows machine was available); `clients/testdata/windows-checklist.md` is the manual run it must pass before live testing. |
| D15 | **Commands are templated from the request**, on the server, from `Host` and TLS state (or `X-Forwarded-Proto`, the same rule as `CheckWebSocketOrigin`). The UI readdresses only the **host** to `window.location` when the server saw the same scheme: `-k`, the PowerShell certificate callback, `--tls-verify-certificate`, the notes and the fingerprint all follow the scheme, and re-templating them in the browser would duplicate the server. When the schemes differ (a TLS-terminating proxy that does not send `X-Forwarded-Proto`) the panel shows the commands as the server templated them and warns the operator to configure the header. The cleartext, fingerprint and wstunnel-unverified bullets follow the scheme of the command shown, not the page's; a readdressed https command carries a fingerprint caveat instead of the pin promise, since the script pins the device leaf and the proxy at the new host may present another certificate. The certificate fingerprint is templated with the commands. |
| D16 | **wstunnel v10.7.1 is pinned in Mode B commands**, URL and SHA-256 per asset from the upstream release `checksums.txt`; all platforms verify the hash before running; `amd64`/`arm64` detected at paste time. `--tls-verify-certificate` is added only when the operator has installed a CA-signed certificate, which is inferred from the served leaf certificate (issuer differs from subject) since no config flag records it; otherwise the UI states that the wstunnel transport is unauthenticated toward the NanoKVM (wstunnel has no fingerprint pin). |
| D17 | **Bridge and exit are mutually exclusive on the gadget NIC.** Exit's enable refuses when any of: bridge last-known-good `Enabled`, `ReadUplink()=="br0"`, `/sys/class/net/br0` exists, `/boot/rndis.nodhcpd` exists, and names which. `bridge.Manager.Enable` symmetrically refuses when `exit.AnyEnabled()` (any `/etc/kvm/exit/*.json` with `enabled: true`). |
| D18 | **Enabled state is the `enabled` field of the slot config**, with a `pending` field armed during the enable transaction (D23); a slot with `pending: true` is disabled at boot. One script serves every slot. `S94exit` is installed by `new_app_init` (`system_init.cpp`, next to S29bridge) and refreshed together with `S30rndis` by `exit.Reconcile()` at every server start; `archive.go` requires `system/init.d/S94exit` in every package. `EnsureBinary` keeps `/etc/kvm/bin/.<name>.seed` (sha256 of the seed) and re-extracts when the seed changes unless `.<name>.custom` exists. |
| D19 | Status is polled by the UI every 3 s. Per-slot in-memory state: tunnel state, peer, `previousPeer`, `peerChangedAt`, connect time, last connected time, bytes, probe result, `dns.redirected`. `downstream` carries `forward`, `routing`, `tun` (carrier), `hev`, `wstunnel`, `dns`, `nat` from `S94exit status` (which also prints `nic`, `gw` and `dns_redirected`), plus `nic.ifname` as resolved. `lastConnectedAt` is persisted to `/etc/kvm/exit/<n>.state.json`. |
| D20 | **Probe.** While an exit is connected, every 30 s the server dials `1.1.1.1:443` through the slot's SOCKS front door with a 5 s timeout and records reachability and latency. The DNS forwarder's success rate is a second signal. Nothing is probed when no exit is connected. |
| D21 | **Destination policy.** Always denied: loopback, unspecified, link-local (`169.254/16`, `fe80::/10`), multicast, `198.18.0.0/15`. Denied unless the slot's `allowPrivate` (default false): `10/8`, `172.16/12`, `192.168/16`, `100.64/10`, `fc00::/7`. Enforced three times: in `EXIT<n>` (immediate ICMP admin-prohibited to the consumer), in the front door at CONNECT and in the mux at OPEN (SOCKS `0x02`), and in the native clients. The exit device's LAN is refused, not silently proxied. The panel mirrors the same prefixes when a resolver (D3) is typed: an always-denied one is refused inline, a private one unless `allowPrivate` is on, so a saved resolver is never one the front door would refuse; the server's own check stays `dive,ip`. |
| D22 | **IPv6.** This release carries IPv4 only. The forwarder filters AAAA (D3), nothing v6 is forwarded from the gadget NIC (D6), and `nexit/1` parses atyp 4 but answers `0x08` without a v6 socket. A dual-stack consumer can still reach literal-IPv6 destinations through its own other uplink; the spec says so. Full v6 (hev `tunnel.ipv6`, RA, `ip -6 rule`, `ip6tables -t nat`) is a follow-up. |
| D23 | **Enable transaction.** Refuse-checks (D17, `NetworkProtocol(ctx)` non-empty) → seeds → write `hev.yml`, `wstunnel-restrict.yml` → config `enabled:false, pending:true` → `S94exit start <slot>` then `S94exit status <slot>` → start front door and forwarder (bind errors are fatal) → write `gadget.route`, flip `enabled:true`, clear `pending` → `S30rndis restart` → `Rebind` (best effort, D5). Any failure runs `S94exit stop <slot>`, removes the marker, leaves `enabled:false` and returns the step's error as `message`. Disable removes the marker and flips `enabled:false` first, then `S30rndis restart`, `S94exit stop <slot>`, `Rebind`, so a crash mid-way converges to off. A mode change on an enabled slot is applied live by `SetConfig` (env and restrict yaml rewritten, listeners rebuilt, `S94exit start` starts or stops the wstunnel server by `MODE`), which drops the connected exit; the UI allows it and says so. An MTU change likewise runs `S94exit stop` + `start`, since hev reads its MTU at start. |
| D24 | **Memory budget.** hev.yml `misc`: `max-session-count: 256`, `tcp-buffer-size: 16384`, `udp-recv-buffer-size: 65536`, `task-stack-size: 20480`, `connect-timeout: 12000`; hev runs under `ulimit -v 65536` (KiB, `HEV_AS_KB` in the S94exit environment overrides it) in the subshell S94exit starts it from, since BusyBox has no `prlimit`; untested on hardware, and if hev fails to start with 256 sessions the cap is raised. Mux: 128 KiB per-stream window, 4 MiB connection window, 256 streams per slot (see `nexit/1`). Target total ≈ 24 MB across hev, mux and DNS; RSS at 256 sessions is measured in the live checklist and wstunnel's RSS is shown in status like the tunnel page. |
| D25 | **Rebind hooks.** `presentation.Manager.OnRebind` becomes a subscriber list (bridge's `ReattachGadget` stays registered). The exit manager subscribes and, after every rebind and after `Attach`: re-resolves `NIC()` (only after `UDCBound()`, which presentation returns as `(bool, error)` and the exit manager wraps so an error reads as unbound, rejecting `(unnamed net_device)`), writes it to `/etc/kvm/exit/<n>/nic`, runs `S94exit start <slot>`, runs `S30rndis start` if the NIC has no `10.x` address, and rebinds the DNS forwarder to the current address. |
| D26 | **Kernel prerequisites**, verified `=y` in `LicheeRV-Nano-Build` `nanokvm-custom` for `sg2002_licheervnano_sd` and `_sd_minimal`: `TUN`, `IP_ADVANCED_ROUTER`, `IP_MULTIPLE_TABLES`, `NF_CONNTRACK`, `NF_NAT`, `NF_TABLES_IPV4`, `NFT_NAT`, `NFT_MASQ`, `NFT_REJECT`, `NFT_COMPAT` (needed for `-j TCPMSS`/`-j REJECT` under iptables-nft), `IP_NF_FILTER`, `IP_NF_NAT`, `IP_NF_TARGET_MASQUERADE`, `IP_NF_TARGET_REJECT`, `NETFILTER_XT_TARGET_TCPMSS`, `NETFILTER_XT_MATCH_CONNTRACK`, `IP6_NF_FILTER`, `IP6_NF_TARGET_REJECT`. Whether the upstream kernel has the policy-routing and nft_compat options is unverified; the feature is documented as requiring the custom kernel until it is. S94exit fails loudly (`nat:false`, log) when any rule insertion fails. |
| D27 | **Packaging.** Makefile `exit` target stages `build/exit/hev-socks5-tunnel` (musl cross toolchain, `$(RISCV_ARCH_FLAGS)` on CFLAGS and LDFLAGS, `-static`, stripped) and gzips `-9 -n` into `kvmapp/exit/hev-socks5-tunnel.gz`; `package` depends on `tunnels passthrough exit`. `package.sh`: `require_file`, `require_riscv64`, `require_fresh … third_party/hev-socks5-tunnel "make exit"`, `require_same_gz`. `package.yml`: hev submodule rev in the cache key and an explicit `make exit` step. `.gitignore`: `kvmapp/exit/*.gz`. |
| D28 | **What the UI says.** Above the commands: the command downloads a script from this NanoKVM and runs it as your user; it contains a secret that controls the target's internet and will be saved in shell history; the exit connects only after verifying certificate fingerprint `<fp>`; anything this machine can reach becomes reachable from the target. A "view script" link, live only while the slot is enabled and not mid-transaction (the gate answers anything else with the counted 404 of D10, and a 404 is reported as the gate refusing, not as a fetch failure), and a regenerate hint after first connect. The security section states that the consumer already reaches the UI, SSH and VNC on the gadget address and that this feature does not change that. |

## Data model

`/etc/kvm/exit/<n>.json`, mode 0600, written atomically (`utils.WriteFileAtomic`):

```json
{
  "slot": "0",
  "enabled": true,
  "pending": false,
  "mode": "native",
  "token": "k7m2p9vx",
  "tokenCreatedAt": "2026-09-15T20:00:00Z",
  "nic": "gadget",
  "dns": ["1.1.1.1", "8.8.8.8"],
  "mtu": 1280,
  "allowPrivate": false,
  "pinPeer": false
}
```

`mode` is `native` (Mode A) or `wstunnel` (Mode B). Derived, never stored: ports, tun name, table id, rule prefs, chain names, tun address, paths.

Other files: `/etc/kvm/exit/<n>/hev.yml`, `/etc/kvm/exit/<n>/wstunnel-restrict.yml`, `/etc/kvm/exit/<n>/env` (below), `/etc/kvm/exit/<n>/nic` (last resolved gadget ifname, D25), `/etc/kvm/exit/<n>.state.json` (`lastConnectedAt`, `previousPeer`, `peerChangedAt`), `/etc/kvm/exit/gadget.route` (D4, content: slot id).

`/etc/kvm/exit/<n>/env` is rendered by the server from `<n>.json` on every config write and sourced by `S94exit`, so the shell never parses JSON. Every value is shell-safe by construction: the flags are `0|1`, `MODE` is the enum, `NIC` is filtered to `[A-Za-z0-9_.-]` (and is only a fallback; the script resolves the live NIC from configfs first), and `REJECT4` is rendered from the same Go `Policy` the front door enforces, so the chain and the SOCKS layer cannot disagree about D21. The token is never in it.

```sh
SLOT=0
ENABLED=1
PENDING=0
MODE=native
ALLOW_PRIVATE=0
MTU=1280
NIC=usb0
REJECT4="0.0.0.0/8 127.0.0.0/8 169.254.0.0/16 198.18.0.0/15 224.0.0.0/3 10.0.0.0/8 100.64.0.0/10 172.16.0.0/12 192.168.0.0/16"
``` Seeds: `/kvmapp/exit/hev-socks5-tunnel.gz`; extracted binary `/etc/kvm/bin/hev-socks5-tunnel` with `/etc/kvm/bin/.hev-socks5-tunnel.seed`. Logs `/tmp/exit<n>-hev.log`, `/tmp/exit<n>-wstunnel.log`. Pidfiles `/var/run/exit<n>-hev.pid`, `/var/run/exit<n>-wstunnel.pid`. Lock `/var/run/exit<n>.lock`.

`hev.yml` (rendered, keys exactly these):

```yaml
tunnel:
  name: exit0
  mtu: 1280
  ipv4: 198.18.0.1
socks5:
  port: 10800
  address: 127.0.0.1
  udp: 'udp'
misc:
  log-file: /tmp/exit0-hev.log
  log-level: warn
  connect-timeout: 12000
  tcp-buffer-size: 16384
  udp-recv-buffer-size: 65536
  task-stack-size: 20480
  max-session-count: 256
```

`wstunnel-restrict.yml`:

```yaml
restrictions:
  - name: exit0
    match:
      - !PathPrefix "^exit0$"
      - !Authorization "^Bearer k7m2p9vx$"
    allow:
      - !ReverseTunnel
        protocol: [Socks5]
        port: [10820]
        cidr: [127.0.0.1/32]
```

## Processes per slot

| Process | Binary | Started by | Bound to |
|---|---|---|---|
| `hev-socks5-tunnel` | `/etc/kvm/bin/hev-socks5-tunnel`, seed `/kvmapp/exit/hev-socks5-tunnel.gz` | `S94exit start <n>` (boot, watchdog, post-attach) | attaches to the persistent tun `exit<n>`; dials `127.0.0.1:10800+n` |
| `wstunnel server` (Mode B only) | `/etc/kvm/bin/wstunnel` | `S94exit start <n>` | `127.0.0.1:10810+n`; reverse listener `127.0.0.1:10820+n` |
| SOCKS front door | in NanoKVM-Server | manager start for every enabled slot | `127.0.0.1:10800+n` |
| DNS forwarder | in NanoKVM-Server | manager start; re-bound on every rebind (D25) | `<gadget ip>:53` UDP and TCP |
| WS endpoint | in NanoKVM-Server | route | `/exit/<n>/native`, `/exit/<n>/*rest` |

The SOCKS front door and the DNS forwarder are unconditional while the slot is enabled. The front door answers SOCKS5 immediately and fails the CONNECT with `0x03` in under a millisecond when no exit is attached, so a consumer with no exit sees connection refused, not a hang. Nothing new listens on `0.0.0.0`; the test suite asserts it.

## The `nexit/1` protocol (Mode A)

### Transport

One WebSocket, upgraded at `GET /exit/<n>/native` with `Authorization: Bearer <token>`. No `Sec-WebSocket-Protocol` is used (`/` is not a token character); the version is negotiated in HELLO/WELCOME. Exactly one nexit frame per WS binary message: a frame split across messages, two frames in one message, or a text message closes the session. Clients must reassemble WS fragmentation (the .NET Framework `ClientWebSocket` fragments large sends); the kvm never emits the 8-byte (127) length form. Client frames are masked per RFC 6455 and clients verify `Sec-WebSocket-Accept`. Both sides enforce a read limit of 16 KiB + 64 per message. The exit uses exactly the scheme templated into its script and never switches scheme on failure.

### Frames

Every frame is `type:u8, stream:u32be, payload`. Stream ids are allocated by the kvm, odd, and never reused within a session; stream `0` is the connection itself.

| Type | Name | Direction | Payload |
|---|---|---|---|
| 0x01 | HELLO | exit → kvm | `version:u8=1, flags:u8, hostname: u8 len + utf8, os: u8 len + utf8` (one length byte each, so at most 255 bytes; longer values are truncated by the sender). `flags` bit 0: the exit has an IPv4 default route. |
| 0x02 | WELCOME | kvm → exit | `version:u8=1, streamWindow:u32be=131072, connWindow:u32be=4194304, maxStreams:u16be=256` |
| 0x10 | OPEN | kvm → exit | `proto:u8 (1=tcp, 2=udp)`; for tcp then `atyp:u8 (1=ipv4, 4=ipv6), addr, port:u16be`; for udp nothing more |
| 0x11 | OPENED | exit → kvm | tcp: `atyp, bound addr, port`; udp: empty |
| 0x12 | OPEN_FAIL | exit → kvm | `reason:u8` (table below) |
| 0x20 | DATA | both | tcp: bytes; udp: `atyp, addr, port, datagram` |
| 0x21 | EOF | both | none; half close of the sender's direction |
| 0x22 | RST | both | optional `reason:u8`; abort |
| 0x30 | WINDOW | both | `increment:u32be`; on stream 0 the connection window |

Payloads are at most 16 KiB. Domain destinations never appear: hev sends literal IPs and the front door answers SOCKS domain CONNECTs with `0x08`. This release carries IPv4 destinations; atyp 4 must be parsed and is answered `0x08` by an exit without an IPv6 socket.

### Flow control

Credit is additive (HTTP/2 `WINDOW_UPDATE` semantics) and applies to TCP streams only. Each TCP stream starts with `streamWindow` bytes of credit in each direction and the connection starts with `connWindow`; DATA consumes both; the receiver sends WINDOW for a stream once it has passed at least half of `streamWindow` on to its socket, and for stream 0 once it has consumed half of `connWindow`. A sender with no credit waits. UDP streams are not credited: each side keeps at most 32 unsent datagrams per stream and drops the oldest; a datagram larger than 16 KiB minus the header is dropped by the sender. The kvm fails a SOCKS CONNECT locally with `0x01` when `maxStreams` streams are open. Per-stream throughput is bounded by `streamWindow / RTT` (≈2.5 MB/s at 50 ms); that is an accepted limit.

### Stream lifecycle

OPEN is answered by exactly one of OPENED or OPEN_FAIL. DATA on a stream that has not been OPENED, and OPEN on a live id, are session errors (close the WS). After OPENED each direction is open independently until that direction's EOF. After sending EOF a side keeps reading and keeps sending WINDOW. DATA received after that peer's EOF is a protocol error answered with RST. A stream is released when both EOFs have been exchanged or when either side has sent or received RST. Frames for unknown ids are ignored (except OPEN on a fresh odd id, which is normal), so a RST racing in-flight DATA is harmless to the stream; the connection window is another matter. Credit is additive and the sender debited `connWindow` when the DATA left, so the receiver still accounts TCP DATA for a released id on stream 0 (the rule RFC 7540 §6.9 has for closed streams), otherwise every aborted stream leaks up to one in-flight window of the sender's credit for good. UDP DATA is uncredited either way, so each side remembers the proto of its recently released ids. Likewise a side that aborts a stream drops what it still had queued for its socket and credits it, including after a clean two-way EOF when the socket side goes away before draining. On WS loss the kvm immediately RSTs every stream toward hev.

Exit-side half close: Python `sock.shutdown(SHUT_WR)`, Perl `shutdown($s, 1)`, PowerShell `$tcp.Client.Shutdown([Net.Sockets.SocketShutdown]::Send)`; in all three the read loop continues until the socket returns 0 bytes, then sends EOF.

### Reasons and timeouts

`reason` is the SOCKS5 REP byte, reused in RST: `0x01` general, `0x02` not allowed by ruleset (destination policy, D21), `0x03` network unreachable, `0x04` host unreachable, `0x05` connection refused, `0x06` TTL/timeout, `0x07` command not supported, `0x08` address type not supported. Mapping: Python `OSError.errno` {ECONNREFUSED→5, EHOSTUNREACH→4, ENETUNREACH→3, ETIMEDOUT→6, EADDRNOTAVAIL/EAFNOSUPPORT→8}; Perl `$!{…}` via `Errno` the same; PowerShell `SocketErrorCode` {ConnectionRefused→5, HostUnreachable→4, NetworkUnreachable→3, TimedOut→6, AddressFamilyNotSupported→8}.

Connects on the exit are non-blocking (Python `connect_ex` + selector + `SO_ERROR`; Perl `IO::Socket::IP->new(Blocking=>0)` + `can_write` + `sockopt(SO_ERROR)`; PowerShell `ConnectAsync` task polled in the loop) with a 6 s budget. The kvm runs an 8 s OPEN timer that yields `0x06`. hev's `misc.connect-timeout` is 12000 ms so it always outlives the kvm timer.

### UDP

The front door implements SOCKS UDP ASSOCIATE as a real loopback relay: per association it allocates `127.0.0.1:0`, returns it as BND.ADDR, binds the association to the source of the first datagram from hev, and maps it 1:1 to one nexit UDP stream. The association is torn down (RST) when hev's control TCP connection closes or after 60 s idle (hev's `udp-read-write-timeout`). The exit uses one unconnected UDP socket per stream (`sendto` / `IO::Socket::IP` / `UdpClient.Send(buf, len, IPEndPoint)`) and records the source of every reply in the DATA header (full-cone). In Mode B the front door relays the SOCKS bytes verbatim so wstunnel's own UDP server address reaches hev.

### Session and keepalive

HELLO is the first frame and must arrive within 5 s of the upgrade; WELCOME closes any previous session on the slot (D12). Keepalive is at the WS level only: the kvm pings every 20 s and closes after three misses (read deadline extended in the pong handler); the exit pings every 30 s (PowerShell `Options.KeepAliveInterval`; Perl/Python opcode 0x9) and, if it receives no frame of any kind for 75 s, closes and reconnects with backoff 1, 2, 4 … 30 s. Both sides apply the destination policy (D21) at OPEN.

### Go mux architecture

One reader goroutine that only parses frames and appends DATA to a per-stream bounded queue (bounded by credit); one writer goroutine fed by a channel (gorilla allows one concurrent writer); per-stream goroutines that drain queues into the hev-facing socket and emit WINDOW. A dedicated `websocket.Upgrader{ReadBufferSize: 32<<10, WriteBufferSize: 32<<10}` with `conn.SetReadLimit(16<<10 + 64)`; the shared `ws/service.go` upgrader (1 KiB buffers, no read limit) is not reused. The reader never blocks on a hev socket.

### Client architecture

Each client is one file of roughly 600 lines with a reconnect loop and no dependencies beyond its runtime, and runs the same Go `httptest` conformance suite (PowerShell via a Windows checklist item).

- **PowerShell 5.1**: one outstanding `ReadAsync` and at most one `WriteAsync` per TCP stream plus one `ReceiveAsync` for the WS, polled with `[Threading.Tasks.Task]::WaitAny($tasks, 250)`; kvm DATA is never written synchronously: it is queued per stream and drained one `WriteAsync` at a time, with the stream's credit returned only when the write completes, so the kvm's stream window bounds the queue and a destination that stops reading holds at most one window of bytes and stalls no other stream (the same shape as Python's `wbuf`/`EV_WRITE` and Perl's); destination writes have no timeout, the bytes sit until the socket errors or the kvm sends RST; every `SendAsync` awaited synchronously (one in flight allowed); `ReceiveAsync` looped on `EndOfMessage` with a 32 KiB buffer; big-endian fixups via `[BitConverter]`/`[Buffer]::BlockCopy`; a compiled `Add-Type` class holds the fingerprint-pinning `RemoteCertificateValidationCallback`; `SecurityProtocol = 'Tls12,Tls13'` in try/catch. Expect 5-20 MB/s.
- **Perl 5**: `IO::Select` over one non-blocking `IO::Socket::SSL` (`SSL_verify_mode => SSL_VERIFY_NONE, SSL_fingerprint => 'sha256$…'`) or `IO::Socket::IP`, plus `IO::Socket::IP` streams; a receive-buffer state machine for WS frames; `pending()`-aware reads with `SSL_WANT_READ/WRITE` handling; `syswrite` partial-write loops; masking as one string XOR.
- **Python 3**: `selectors.DefaultSelector` over one non-blocking `ssl.SSLSocket` (`CERT_NONE` then a `getpeercert(binary_form=True)` SHA-256 compare before any byte is trusted) plus stream sockets; same state machine; `SSLWantReadError`/`pending()` rule; `int.from_bytes` XOR masking.

## HTTP surface

Token-gated with `Authorization: Bearer <token>` (D10), no JWT, outside `/api`. Every rejection is a 404 byte-identical to gin's default (tested header-for-header):

```
GET  /exit/:slot/native        nexit/1 WebSocket upgrade (Mode A)
GET  /exit/:slot/client.sh     sh bootstrap (picks python3 or perl)
GET  /exit/:slot/client.ps1    PowerShell client
GET  /exit/:slot/client.pl     Perl client
GET  /exit/:slot/client.py     Python client
GET  /exit/:slot/*rest         Mode B: Upgrade: websocket requests whose rest ends in /events are
                               proxied to wstunnel server as /exit<n>/<rest>; anything else 404
```

Scripts are served `text/plain; charset=utf-8`, fully templated (scheme, host, slot, token, certificate SHA-256), with no argv.

Admin API, JWT and `RequireRole(admin)`, under `/api/extensions/exit`, in the `{code,msg,data}` envelope like every other route:

```
GET    /slots                          list slots with status
GET    /:slot/status
GET    /:slot/config                   config without the token (token comes with status)
POST   /:slot/config                   { mode, dns?, mtu?, allowPrivate?, pinPeer? }
POST   /:slot/enable
POST   /:slot/disable
POST   /:slot/token/regenerate
POST   /:slot/disconnect               drop the active exit
GET    /:slot/commands                 templated commands for both modes and three platforms, plus fingerprint
GET    /:slot/logs                     hev + wstunnel log tails, token-shaped values redacted
```

Status response:

```json
{
  "slot": "0", "enabled": true, "pending": false, "mode": "native", "token": "k7m2p9vx",
  "tunnel": "connected",
  "peer": { "addr": "203.0.113.7", "hostname": "robs-laptop", "os": "darwin", "transport": "native" },
  "previousPeer": null, "peerChangedAt": null,
  "connectedAt": "…", "lastConnectedAt": "…", "uptimeSeconds": 812,
  "nic": { "ifname": "usb0", "up": true, "protocol": "ncm", "address": "10.12.34.1/24" },
  "downstream": { "forward": true, "routing": true, "tun": true, "hev": true, "wstunnel": true, "dns": true, "nat": true },
  "upstream": { "reachable": true, "latencyMs": 41, "checkedAt": "…" },
  "dns": { "queries": 120, "failures": 0, "redirected": 3 },
  "bytes": { "up": 1234, "down": 5678 },
  "message": ""
}
```

`tunnel` is one of `disconnected`, `connecting`, `connected`. `connecting` means an upgrade has been accepted but the HELLO (Mode A) or the reverse listener (Mode B) has not appeared yet. `downstream` is the boolean vector `S94exit status <n>` reports; `nic.ifname` is the value resolved after the last attach or rebind.

## Enable transaction

As D23. In order: refuse-checks (D17; `NetworkProtocol(ctx)` non-empty, else "turn on the USB Network Adapter under Network", the name the UI gives that setting) → `EnsureBinary` for hev and, in Mode B, wstunnel → write `hev.yml` and `wstunnel-restrict.yml` → config `enabled:false, pending:true` → `S94exit start <n>` then `S94exit status <n>` → start the SOCKS front door and the DNS forwarder (bind errors are fatal) → write `gadget.route`, flip `enabled:true`, clear `pending` → `S30rndis restart` → `Rebind` (best effort, D5).

Any failure runs `S94exit stop <n>`, stops the in-process listeners, removes the marker, leaves `enabled:false`, and returns the failing step's error as `message`.

Disable: remove the marker and flip `enabled:false` first, then `S30rndis restart`, stop the listeners, `S94exit stop <n>`, `Rebind`. A crash midway converges to off at the next boot or watchdog tick: boot skips disabled slots, and the watchdog runs `S94exit status <n>` for a disabled slot and `S94exit stop <n>` when that still reports hev, routing or NAT up. The consumer's NIC comes back after the cycle with a lease that carries no router and no DNS, exactly the stock state.

Both transactions (and `SetConfig`) run on a context detached from the HTTP request: gin cancels the request context when the client goes away, and `exec.CommandContext` then refuses to start a command, which would leave `enabled:false` recorded with hev, the tun and the chains still up. The per-command 60 s timeout in `execRunner` is the only bound.

## Boot

`S03usbdev` builds the composite but does not bind the UDC, so at `S30rndis` and `S94exit` time the gadget netdev does not exist yet (`ifname` reads `(unnamed net_device)`), unless the 30 s fallback fired. `S94exit start` at boot therefore brings up the NIC-independent half only: sysctls, the persistent tun, the routing table and the daemons; the `iif <gadget>` rules, chains and the DNS bind wait. The server at `S95` binds the UDC in `Attach`, and the post-attach hook (D25) runs `S30rndis start` (address, udhcpd with the gateway options because `gadget.route` exists) and `S94exit start <n>` (rules, chains, DNAT), then binds the DNS forwarder. The consumer sees the NIC when the gadget binds and receives a lease with gateway and DNS within seconds; names SERVFAIL until an exit connects, then everything flows. This ordering must be confirmed on hardware first: `ip addr show usb0` and `pgrep -a udhcpd` after a clean, unbridged boot. `exit.GetManager().Init()` (script install, config load, `S94exit start`/`status` per enabled slot) runs inside the router's 15 s "usb attach" startup budget; it is measured nowhere yet, and if it eats the budget on hardware Init moves ahead of Attach and only OnAttach stays in the hook.

## Files

Backend:

| Path | Action |
|---|---|
| `server/proto/exit.go` | new: types, states, requests, responses |
| `server/service/exit/slot.go` | new: slot id, derived names, ports, prefs, chains, paths |
| `server/service/exit/config.go` | new: atomic load and save, token generation, state file |
| `server/service/exit/manager.go` | new: per-slot runtime, enable/disable transactions (D23), watchdog converge, rebind subscriber (D25), bridge gate (D17) |
| `server/service/exit/render.go` | new: `hev.yml`, `wstunnel-restrict.yml` rendering |
| `server/service/exit/downstream.go` | new: runs `S94exit`, `S30rndis`, parses `status`, installs/refreshes both scripts into `/etc/init.d` |
| `server/service/exit/policy.go` | new: destination policy (D21) shared by front door and mux |
| `server/service/exit/socks.go` | new: SOCKS5 front door, CONNECT + UDP ASSOCIATE relay, backend dispatch, Mode B byte relay |
| `server/service/exit/mux.go`, `frame.go`, `udp.go` | new: `nexit/1` server side, wire format and credit, SOCKS UDP ASSOCIATE relay |
| `server/service/exit/wire.go` | new: `newComponents`, the one place the manager's dataplane is built from the real constructors |
| `server/service/exit/counter.go`, `install.go`, `clients_embed.go` | new: byte counter, init-script install, `go:embed` of exactly the four clients |
| `server/service/exit/nexittest.go`, `conformance_test.go` | new: in-process fake exit (the mux tests' oracle); the tagged conformance suite running the Python and Perl clients |
| `server/service/exit/wsproxy.go` | new: Mode B reverse proxy with connection tracking, header hygiene, supersede |
| `server/service/exit/dns.go` | new: forwarder over SOCKS TCP, AAAA filter, counters |
| `server/service/exit/probe.go` | new: reachability probe |
| `server/service/exit/ratelimit.go` | new: per-source token attempt limiter |
| `server/service/exit/commands.go` | new: command templating, pinned wstunnel table with SHA-256 per asset, fingerprint |
| `server/service/exit/clients/{client.sh,client.ps1,client.pl,client.py}` | new: embedded native clients (`go:embed`), templated; `clients/README.md` and `clients/testdata/{mock_kvm.py,host_tests.py,windows-checklist.md}` are not embedded |
| `server/service/exit/service.go` | new: gin handlers |
| `server/router/exit.go` | new: both route groups |
| `server/router/router.go` | modify: register, post-attach hook |
| `server/service/extensions/tunnel/wrapper.go` | modify: export `EnsureBinary(name)` with seed-hash staleness (D18) |
| `server/service/presentation/manager.go` | modify: `OnRebind` becomes a subscriber list (D25) |
| `server/service/bridge/bridge.go`, `bridge/apply.go` | modify: register via the list; refuse `Enable` when `exit.AnyEnabled()` (D17) |
| `server/service/application/archive.go` | modify: require `system/init.d/S94exit` |

Device and build:

| Path | Action |
|---|---|
| `kvmapp/system/init.d/S94exit` | new: converge script (D8), `start|stop|status [slot]` |
| `kvmapp/system/init.d/S30rndis` | modify: `opt router`/`opt dns` when `/etc/kvm/exit/gadget.route` exists (D4); `(unnamed net_device)` treated as no NIC |
| `support/sg2002/kvm_system/main/lib/system_init/system_init.cpp` | modify: copy `S94exit` next to `S29bridge` |
| `third_party/hev-socks5-tunnel` | submodule; fork may set lwip `netif->mtu` from `tunnel.mtu` (D2) |
| `Makefile` | `exit` target (D27); `package` depends on it |
| `scripts/package.sh` | `require_file`, `require_riscv64`, `require_fresh`, `require_same_gz` for the seed |
| `.gitignore` | `kvmapp/exit/*.gz` |
| `.github/workflows/package.yml` | `make exit`, hev rev in the cache key |

Frontend:

| Path | Action |
|---|---|
| `web/src/api/extensions/exit.ts` | new |
| `web/src/pages/desktop/menu/settings/exit/index.tsx` | new: renders one `ExitSlot` per slot from `/slots` |
| `web/src/pages/desktop/menu/settings/exit/slot.tsx` | new: the repeatable unit, 3 s polling |
| `web/src/pages/desktop/menu/settings/exit/status.tsx` | new |
| `web/src/pages/desktop/menu/settings/exit/token.tsx` | new: always visible, copy, regenerate |
| `web/src/pages/desktop/menu/settings/exit/commands.tsx` | new: mode tabs, platform sub-tabs, copy buttons, warnings (D28) |
| `web/src/pages/desktop/menu/settings/exit/types.ts`, `state.ts` | new |
| `web/src/pages/desktop/menu/settings/exit/advanced.tsx`, `logs.tsx` | new: the Advanced collapse (DNS, MTU, allowPrivate, pinPeer) and the log tails, fetched on expand |
| `web/src/lib/clipboard.ts` | new: copy with the `execCommand` fallback an http-served page needs |
| `web/src/pages/desktop/menu/settings/index.tsx` | modify: one tab `exit` |
| `web/src/i18n/locales/*.ts` | modify: `settings.exit.*` in all 24 |

## Testing

Unit, in Go: config round trip, token alphabet and length, rate limiter windows, SOCKS5 handshake parsing, `nexit/1` framing and flow control against an in-process fake exit, DNS forwarder against a fake upstream over a fake SOCKS, command templating for every platform and scheme, path rewrite for the wstunnel proxy, restrict yaml rendering, hev.yml rendering.

Integration, on the host: `go test -race -tags conformance -run Conformance ./service/exit/` runs the Python and Perl clients as subprocesses against the real mux and front door (echo, 20 MB both ways with sha256, half close, OPEN_FAIL 0x05, UDP ASSOCIATE echo, supersede); it needs a non-loopback IPv4 address on the host because the policy denies 127/8 everywhere. PowerShell is exercised under `pwsh` on the host against `clients/testdata/mock_kvm.py` with `host_tests.py` (including `stalled-sink`, the head-of-line check: a destination that accepts and never reads must not stall OPEN, DATA or WINDOW on other streams) and on Windows PowerShell 5.1 from `clients/testdata/windows-checklist.md`. The Mode B proxy against a real `wstunnel` binary and end-to-end `curl --socks5` through hev remain live-test items.

Live, on hardware: everything in the task's TESTING list, driven from the checklist in `docs/exit-live-test.md` written alongside the implementation.

## Risks

1. `hev-socks5-tunnel` has never run on this kernel or CPU. First check: `hev-socks5-tunnel <conf>` brings up `exit0`, `ip addr` shows `198.18.0.1`, and `curl --interface exit0` fails fast with no exit.
2. Gadget pull-up cycle on enable drops HID for about two seconds. Deliberate, documented in the UI.
3. SOCKS UDP through wstunnel's reverse SOCKS is the least exercised path. DNS does not depend on it (D3); QUIC falls back to TCP.
4. RAM. hev-socks5-tunnel is small; wstunnel server is the same binary already measured. The Go side bounds buffering per stream to one window.
5. The consumer keeps our default route even when it has another uplink. That is the point of the feature and also why the options are only handed out while enabled.

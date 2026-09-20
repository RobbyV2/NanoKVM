# Exit tunnel: hardware live-test checklist

Spec: `docs/superpowers/specs/2026-09-15-exit-tunnel-design.md`. Nothing in that spec has run on a
device before this checklist is executed. Run it top to bottom; tick or file every step with the
capture list at the end.

## Conventions

Three machines. Keep a terminal open on each.

| Name | Machine | Shell prompt in this doc |
|---|---|---|
| NanoKVM | the device, `ssh root@<mgmt-ip>` over eth0/Wi-Fi (not over the gadget NIC, which re-enumerates in §12) | `kvm#` |
| Consumer | the host the NanoKVM's USB-C is plugged into | `win>` (PowerShell), `mac$`, `lin$` |
| Exit | the laptop that browses to the UI and runs the pasted command | `exit$` / `exit>` |

Exit and Consumer have to be different machines. Running the pasted command on the consumer makes it
its own exit: its packets leave over the gadget NIC, come back through the tunnel and are handed to
its own routing table. With a real exit also connected the two supersede each other once a second
(§9.3) and the consumer's traffic dies in the churn, which takes any remote management of that
machine with it, including whatever would stop the client again. Give a client started over such a
link its own stop timer before you start it, and keep the NanoKVM's own §8 token regeneration as the
way to lock out a client you can no longer reach.

Slot 0 only. Derived names, verbatim from `server/service/exit/slot.go` and `S94exit`:

| Thing | Value |
|---|---|
| tun | `exit0`, `198.18.0.1/32`, MTU 1280 |
| routing table / rule prefs | table `100`; pref `1000` (`iif $NIC lookup 100`), pref `1001` (`iif $NIC unreachable`) |
| iptables chains | `EXIT0` (filter FORWARD), `EXIT0_NAT` (nat PREROUTING), `EXIT0_MASQ` (nat POSTROUTING) |
| loopback ports | `127.0.0.1:10800` SOCKS front door, `:10810` wstunnel server (Mode B), `:10820` wstunnel reverse listener (Mode B) |
| files | `/etc/kvm/exit/0.json`, `/etc/kvm/exit/0/env`, `/etc/kvm/exit/0/hev.yml`, `/etc/kvm/exit/0/wstunnel-restrict.yml`, `/etc/kvm/exit/0/nic`, `/etc/kvm/exit/0.state.json`, `/etc/kvm/exit/gadget.route` |
| runtime | `/var/run/exit0-hev.pid`, `/var/run/exit0-wstunnel.pid`, `/var/run/exit0-hev.spawns`, `/var/run/exit0-wstunnel.spawns` (spawns since the last stop; `status` prints them as `hev_spawns`, `wstunnel_spawns`), `/var/run/exit0.lock`, `/tmp/exit0-hev.log`, `/tmp/exit0-wstunnel.log` (each cut back in place to its last 64 KiB by the converge above 1 MiB; a daemon started with under 512 KiB free on `/tmp` logs to `/dev/null` and `S94exit start` says so) |
| binaries | `/etc/kvm/bin/hev-socks5-tunnel` (seed `/kvmapp/exit/hev-socks5-tunnel.gz`), `/etc/kvm/bin/wstunnel` |
| scripts | `/etc/init.d/S94exit`, `/etc/init.d/S30rndis` (seeds in `/kvmapp/system/init.d/`) |
| HTTP | token-gated `GET /exit/0/{native,client.sh,client.ps1,client.pl,client.py}` and `/exit/0/*` (Mode B); admin `/api/extensions/exit/0/{status,config,enable,disable,token/regenerate,disconnect,commands,logs}` |
| UI | Settings → Exit (admin only), polls status every 3 s |
| timers | watchdog converge 30 s; probe `1.1.1.1:443` every 30 s while connected; kvm WS ping 20 s, close after 3 misses (read deadline 100 s); HELLO within 5 s; OPEN timer 8 s; client reconnect backoff 1,2,4,8,16,30 s; rate limiter 3 free failures then 1,2,4,… s, cap 15 min |

Set these on the NanoKVM at the start of every ssh session (the gadget NIC is not reliably `usb0`):

```sh
kvm# C=/sys/kernel/config/usb_gadget/g0/configs/c.1
kvm# NIC=$(for f in $C/ncm.* $C/rndis.* $C/eem.*; do [ -r "$f/ifname" ] && cat "$f/ifname"; done | grep -v '[(%]' | head -1); echo "NIC=$NIC"
kvm# GW=$(ip -4 -o addr show dev "$NIC" | awk '{print $4}' | cut -d/ -f1); echo "GW=$GW"
kvm# S=/etc/init.d/S94exit
```

`$GW` is `10.<a>.<b>.1`; the consumer's lease is `10.<a>.<b>.100-200`. `$KVM` below is the host the
exit device reaches the UI at (whatever is in the browser's address bar). `$TOKEN` is the 8-character
token shown in the Exit panel (`[a-z2-9]{8}`, no `0 o 1 l i`).

For §3–§11 the consumer must have **no other uplink** (Wi-Fi off, other Ethernet unplugged) or the
result says nothing about the tunnel. §12 puts it back.

BusyBox `netstat` here rejects `-p` and exits non-zero, which with stderr discarded reads as an empty
result, so every listener question in this doc is answered with `ss -lnupt` (or `ss -lntp`, `ss -ntu`)
in place of `netstat -lntp`. `conntrack` is absent as well, so each step that offers a
`/proc/net/nf_conntrack` fallback takes it.

The Windows consumer here resolves `example.com` to `127.0.0.1`, so every `win>` step that needs
a name it can reach uses `api.ipify.org`, `www.iana.org` or `www.ietf.org` in its place.

Counters: every 30 s the watchdog runs `S94exit start 0`, which replaces `EXIT0`, `EXIT0_NAT` and
`EXIT0_MASQ` in one `iptables-restore -n` transaction, so their `iptables -L … -v -n -x` packet counters **reset every 30 s**.
Read them within a few seconds of generating the traffic. Cumulative numbers are `dns_redirected`
(carried across flushes in `/var/run/exit0.dns_redirected`, cleared on `stop`), `ip -s link show exit0`,
conntrack, and the UI's `bytes`/`dns` fields.

To see the server's own log: it is launched by `S95nanokvm` as `/tmp/server/NanoKVM-Server &` with
`logger.file: stdout` by default (`/etc/kvm/server.yaml`), so its stdout goes nowhere useful. For the
whole run, do once:

```sh
kvm# /etc/init.d/S95nanokvm stop; sleep 2; cd /tmp/server && (./NanoKVM-Server 2>&1 | tee /tmp/server.log) &
```

(or set `logger: {file: /tmp/server.log}` in `/etc/kvm/server.yaml` and
`/etc/init.d/S95nanokvm restart > /root/server.out 2>&1 < /dev/null`). The server handles only INT,
TERM and QUIT and exits on SIGPIPE at its next log line once the pipe it writes to closes, so the
ssh session that runs either command stays open for the whole run, and a restart without the
redirect leaves the new server's stdout and stderr on the ssh channel. The
manager logs as `exit: slot 0 …` and the mux/proxy as `exit0: …`.

---

## 0. Pre-flight, hardware-only questions

Everything here is read-only except 0.5, 0.9, 0.10, 0.13, which create and delete throwaway objects.
Do this on the exact unit and image that will run §1–§12, with the slot still disabled
(`grep '"enabled"' /etc/kvm/exit/0.json` → `false`, or no file yet).

### 0.1 Which image, which kernel

```sh
kvm# cat /etc/kvm/version 2>/dev/null; cat /kvmapp/version 2>/dev/null; uname -a
kvm# test -f /etc/nanokvm-minimal && echo MINIMAL || echo FULL
kvm# ls /boot/usb.* /boot/rndis.* 2>/dev/null
kvm# cat /sys/class/udc/*/state
```

Expected: kernel `5.10.x riscv64`; the sentinel list tells you the profile (`usb.ncm` or `usb.rndis0`
must be present, `usb.disk0` and a UVC sentinel decide §12's scope); UDC `configured` when the
consumer is plugged in. Record `MINIMAL`/`FULL`: the minimal image has legacy iptables only, no nft,
no dnsmasq.

### 0.2 Kernel symbols (D26)

```sh
kvm# for k in TUN IP_ADVANCED_ROUTER IP_MULTIPLE_TABLES NF_CONNTRACK NF_NAT NF_TABLES_IPV4 NFT_NAT NFT_MASQ NFT_REJECT NFT_COMPAT IP_NF_IPTABLES IP_NF_FILTER IP_NF_NAT IP_NF_TARGET_MASQUERADE IP_NF_TARGET_REJECT NETFILTER_XT_TARGET_TCPMSS NETFILTER_XT_TARGET_MASQUERADE NETFILTER_XT_NAT NETFILTER_XT_MATCH_CONNTRACK IP6_NF_IPTABLES IP6_NF_FILTER IP6_NF_TARGET_REJECT IKCONFIG_PROC; do printf '%-36s %s\n' "CONFIG_$k" "$(zcat /proc/config.gz | grep -E "^CONFIG_$k=" | cut -d= -f2)"; done
```

Expected: every line ends in `y`. Pass: no blank and no `m` (nothing `=m` can load: there is no
`modules.dep`). A blank on `NFT_COMPAT` with an nft-backend iptables (0.4) means `-j TCPMSS`/`-j REJECT`
will fail in 0.10 and the feature needs the custom kernel.

### 0.3 `/dev/net/tun` opens

```sh
kvm# ls -l /dev/net/tun; grep -c tun /proc/misc
kvm# ip tuntap add dev tuntest mode tun && ip addr add 198.18.9.1/32 dev tuntest && ip link set tuntest mtu 1280 up && ip -d link show tuntest && cat /sys/class/net/tuntest/carrier; ip link del tuntest
```

Expected: `/dev/net/tun` at `10, 200`, mode `0666` on this image; `1`; the add succeeds, `ip -d link`
shows `tun` and `mtu 1280`, carrier prints `0` (persistent tun with no reader: that is the fail-closed state D6
relies on). Pass: no `Operation not permitted` / `No such device`.

### 0.4 Userland binaries and the iptables backend

```sh
kvm# for b in ip iptables ip6tables nft conntrack udhcpd dnsmasq sysctl flock start-stop-daemon awk tcpdump netstat ss gzip; do printf '%-18s %s\n' "$b" "$(command -v $b || echo MISSING)"; done
kvm# ip -V; readlink -f "$(command -v ip)"
kvm# iptables -V; readlink -f "$(command -v iptables)"
kvm# iptables -S FORWARD; iptables -t nat -S PREROUTING; iptables -t nat -S POSTROUTING
```

Expected: `ip` is iproute2 (`ip utility, iproute2-…`), **not** a BusyBox applet (BusyBox `ip rule`
does not accept `unreachable` and `ip route replace unreachable default …` may fail: S94exit needs
the real one). `iptables -V` names the backend the image carries, legacy or nf_tables, and S94exit
runs on either once 0.10 passes; the FULL image on this unit reports legacy:

```
iptables v1.8.9 (legacy)
/usr/sbin/xtables-legacy-multi
```

`flock`, `start-stop-daemon`, `sysctl`, `awk`, `udhcpd` present, `conntrack` MISSING. Record whether
`tcpdump` exists. Save the three baseline chain listings for §12's before/after comparison.

### 0.5 Who owns port 53, and is dnsmasq configured

```sh
kvm# ls -l /etc/dnsmasq.conf /etc/dnsmasq.d 2>&1; pgrep -a dnsmasq
kvm# ss -lnupt              # who holds :53 and :10800, with the process name
kvm# cat /etc/resolv.conf
```

Expected on a stock unit: no `/etc/dnsmasq.conf`, no dnsmasq process, nothing on `:53`. Once the slot
is enabled the same command shows `:53` held by NanoKVM-Server on the gateway address alone, udp and
tcp on `$GW:53` (`10.163.245.1:53` on this unit), with the SOCKS front door on `127.0.0.1:10800` and
nothing on `0.0.0.0`. If anything listens on `0.0.0.0:53` or `$GW:53`, **the enable in §1 will fail**
at `start listeners: dns forwarder on $GW:53: … address already in use` (the D3 dnsmasq drop-in is not implemented in the server; `grep -rl
dnsmasq server/` is empty). File that with the listener's cmdline before continuing, then stop it
for the run (`/etc/init.d/S80dnsmasq stop`).

### 0.6 `ip_forward` and S98tailscaled

```sh
kvm# sysctl -n net.ipv4.ip_forward
kvm# ls -l /etc/init.d/S98tailscaled /usr/bin/tailscale 2>&1; pgrep -a tailscaled; ls -l /var/run/tailscaled.pid 2>&1
```

On an image that runs tailscaled, `S98tailscaled` carries `sysctl -w net.ipv4.ip_forward=0` in its
stop path and the value reads `0` after `S98tailscaled stop`, which is the condition the watchdog
undoes: re-check in §1 step 1.6 (`forward=1` within 30 s of the stop while the slot is enabled), and
restart tailscaled before going on. This unit ships `/usr/bin/tailscale` and nothing else, no init
script, no daemon and no `/var/run/tailscaled.pid`, so the stop and restart pass is skipped here and
`S94exit stop` is the only thing that touches `ip_forward`.

### 0.7 Gadget netdev timing at a clean boot

This needs ssh over eth0 (sshd is S50, before S95). Have the consumer plugged in, `reboot`, and
reconnect the moment ssh answers, then:

```sh
kvm# C=/sys/kernel/config/usb_gadget/g0/configs/c.1; for i in $(seq 120); do echo "$(cut -d' ' -f1 /proc/uptime) udc=[$(cat /sys/kernel/config/usb_gadget/g0/UDC 2>/dev/null)] ifname=[$(cat $C/ncm.*/ifname $C/rndis.*/ifname 2>/dev/null | tr '\n' ,)] addr=[$(ip -4 -o addr 2>/dev/null | grep -o '10\.[0-9.]*/24' | tr '\n' ,)] udhcpd=[$(ps w | grep '[u]dhcpd' | awk '{print $1}' | tr '\n' ,)] server=[$(pidof NanoKVM-Server)] exit0=[$(cat /sys/class/net/exit0/carrier 2>/dev/null)] hev=[$(cat /var/run/exit0-hev.pid 2>/dev/null)]"; sleep 1; done | tee /tmp/boot-timeline.log
```

Expected sequence (slot disabled at this point): `ifname=[(unnamed net_device)]` and `udc=[]` until
the server starts, then `udc=[4340000.usb]`, ifname becomes `usb0` (or `usb1`), the `10.x.y.1/24`
address and exactly one `udhcpd` appear within a few seconds of `server=[pid]`. Record the uptime at
each transition. Fail: two udhcpd pids; an address with no udhcpd; ifname still `(unnamed …)` 30 s
after the server pid; or (with the slot enabled, repeat in §6) `exit0=[…]`/`hev=[…]` absent before the
server starts, which would mean `S94exit start` at boot did not bring up the NIC-independent half.

Also: `ip addr show usb0` and `ps w | grep '[u]dhcpd'` once, at rest, so the plan's literal question has a
literal answer in the report.

### 0.8 Endpoint budget

```sh
kvm# ls /sys/kernel/debug/usb/ /sys/kernel/debug/usb/4340000.usb/ 2>&1
kvm# cat /sys/kernel/debug/usb/4340000.usb/hw_params 2>/dev/null | grep -iE 'num_dev_ep|num_dev_in_eps|total_fifo|dev_token_q|host_channels'
kvm# cat /sys/kernel/debug/usb/4340000.usb/fifo 2>/dev/null
kvm# ls /sys/kernel/config/usb_gadget/g0/configs/c.1/
```

Expected: the profile's function list (hid.*, ncm.*/rndis.*, mass_storage.*, uvc.*) fits the
IN-endpoint count the hardware reports. Record `num_dev_in_eps` and the FIFO depths verbatim: the
presentation manager's static table (`6 / 5 / [768,512,512,384,128,128]`) is an assumption this
confirms or corrects. If debugfs is not mounted: `mount -t debugfs none /sys/kernel/debug`.

The reading on this unit matches `presentation/capability.go` staticV1 exactly:

```
g_tx_fifo_size[1..6] = 768, 512, 512, 384, 128, 128
num_dev_ep = 7
```

Six of the seven IN endpoints have a dedicated TX FIFO, so six is the usable count, and the five
linked functions charge IN = 6, exactly full, against OUT = 3 of 5.

### 0.9 `ip rule … unreachable` accepted

```sh
kvm# ip rule add pref 1999 iif lo unreachable && ip rule show pref 1999; ip rule del pref 1999 iif lo unreachable
kvm# ip route replace unreachable default metric 4294967295 table 199 && ip route show table 199; ip route flush table 199
```

Expected: `1999: from all iif lo unreachable` and `unreachable default metric 4294967295`. Pass: both
print and both delete cleanly.

### 0.10 `-j TCPMSS`, `-j REJECT`, DNAT, MASQUERADE accepted under this iptables

```sh
kvm# iptables -N EXITTEST && iptables -A EXITTEST -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --set-mss 1240 && iptables -A EXITTEST -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT && iptables -A EXITTEST -j REJECT --reject-with icmp-admin-prohibited && iptables -S EXITTEST; iptables -F EXITTEST; iptables -X EXITTEST
kvm# iptables -t nat -N EXITTEST && iptables -t nat -A EXITTEST -p udp --dport 53 -j DNAT --to-destination 10.99.99.1:53 && iptables -t nat -A EXITTEST -s 10.99.99.0/24 -j MASQUERADE && iptables -t nat -S EXITTEST; iptables -t nat -F EXITTEST; iptables -t nat -X EXITTEST
kvm# ip6tables -N EXITTEST && ip6tables -A EXITTEST -j REJECT --reject-with icmp6-adm-prohibited && ip6tables -S EXITTEST; ip6tables -F EXITTEST; ip6tables -X EXITTEST
```

Expected: every `-A` succeeds and `-S` echoes it back. Fail: `No chain/target/match by that name`
(missing `NFT_COMPAT` or the xt module) — S94exit would then report `nat=0` and log `iptables …: failed`.

### 0.11 Idle RSS baseline

```sh
kvm# free -m; grep -E 'VmRSS|VmSize' /proc/$(pidof NanoKVM-Server)/status
```

Record. At rest NanoKVM-Server holds `VmRSS` near 38 MB behind a ~1.2 GB Go arena in `VmSize`, and
hev-socks5-tunnel near 1.9 MB against its 64 MiB address-space cap. The under-load numbers are taken
in §3 step 3.9 (hev, wstunnel, server at 256 sessions).

### 0.12 Shipped files are present and executable

```sh
kvm# ls -l /kvmapp/exit/ /kvmapp/system/init.d/S94exit /kvmapp/system/init.d/S30rndis /etc/init.d/S94exit /etc/init.d/S30rndis
kvm# cmp /etc/init.d/S94exit /kvmapp/system/init.d/S94exit && cmp /etc/init.d/S30rndis /kvmapp/system/init.d/S30rndis && echo scripts-installed
kvm# ls -l /etc/kvm/bin/; cat /etc/kvm/bin/.hev-socks5-tunnel.seed 2>/dev/null
kvm# /etc/kvm/bin/hev-socks5-tunnel 2>&1 | head -3 || (gzip -dc /kvmapp/exit/hev-socks5-tunnel.gz > /tmp/hev && chmod +x /tmp/hev && /tmp/hev 2>&1 | head -3)
kvm# /etc/kvm/bin/wstunnel --version 2>&1 | head -1
kvm# cat /etc/kvm/exit/0.json | sed 's/"token": *"[^"]*"/"token":"<redacted>"/'; ls -la /etc/kvm/exit/ /etc/kvm/exit/0/ 2>&1
kvm# du -sk /kvmapp/server; ls /kvmapp/server; df -Pk /tmp
```

Expected: `scripts-installed` (the server refreshes both at start); hev prints its usage line, which
proves the static riscv64 binary executes on this CPU (Risk 1); `0.json` exists with `"enabled":
false`, `"pending": false`, `"mode": "native"`, `"nic": "gadget"`, mode `0600`. `hev-socks5-tunnel` may
not be extracted yet: that is fine, the enable does it. `/kvmapp/server` is under 45 MB and holds
only `NanoKVM-Server`, `dl_lib` and `web`: `S95nanokvm` copies all of it into the 80 MB `/tmp` tmpfs
at every boot and restart, and a staged binary or backup left there fills `/tmp`.

### 0.13 Bridge gate is clear

```sh
kvm# ls /sys/class/net/br0 /boot/rndis.nodhcpd 2>&1; cat /etc/kvm/network/l2-uplink 2>/dev/null; grep -o '"enabled": *[a-z]*' /etc/kvm/presentation/network/last-known-good.json 2>/dev/null
```

Expected: none exist / `false`. If any is set, the enable refuses with `the gadget NIC belongs to the
L2 bridge: …` (D17): disable the bridge first.

---

## 1. Mode A commands execute cleanly on each platform and establish the tunnel

### 1.1 Enable

Exit browser: Settings → Exit → slot 0 → switch on → confirm the Popconfirm (it says HID/camera
re-enumerate for ~2 s). The consumer's keyboard/mouse from the UI drop for ~2 s; the consumer's NIC
disappears and reappears.

```sh
kvm# grep -E '"(enabled|pending|mode)"' /etc/kvm/exit/0.json; cat /etc/kvm/exit/gadget.route; cat /etc/kvm/exit/0/env
kvm# $S status 0; echo rc=$?
kvm# ip -d link show exit0; ip addr show exit0; cat /sys/class/net/exit0/carrier
kvm# ip rule; ip route show table 100
kvm# iptables -S EXIT0; iptables -t nat -S EXIT0_NAT; iptables -t nat -S EXIT0_MASQ; ip6tables -S FORWARD | grep "$NIC"
kvm# sysctl net.ipv4.ip_forward net.ipv4.conf.$NIC.route_localnet net.ipv4.conf.exit0.rp_filter net.ipv6.conf.$NIC.disable_ipv6
kvm# netstat -lntp | grep -E ':10800|:53 '; netstat -lnup | grep ':53 '
kvm# ps w | grep '[u]dhcpd'; cat /etc/udhcpd.$NIC.conf
kvm# cat /var/run/exit0-hev.pid; tail -5 /tmp/exit0-hev.log
```

Expected, exactly:

- `"enabled": true`, `"pending": false`, `"mode": "native"`; `gadget.route` contains `0`; env has
  `ENABLED=1 PENDING=0 MODE=native MTU=1280 NIC=$NIC` and a `REJECT4=` list starting `0.0.0.0/8 127.0.0.0/8 169.254.0.0/16 198.18.0.0/15 224.0.0.0/3 10.0.0.0/8 …`.
- `status` prints `forward=1 routing=1 tun=1 hev=1 wstunnel=1 nat=1 nic=$NIC gw=$GW dns_redirected=0`, `rc=0`.
- `exit0`: `mtu 1280`, `tun`, `198.18.0.1/32`, carrier `1` (hev has it open).
- `ip rule` contains `1000: from all iif $NIC lookup 100` and `1001: from all iif $NIC unreachable`,
  both below any Tailscale `52xx` rules; table 100 is exactly `default dev exit0 scope link` and
  `unreachable default metric 4294967295`.
- `EXIT0`: the REJECT lines for each REJECT4 prefix (`-i $NIC -d <p> -j REJECT --reject-with icmp-admin-prohibited`),
  then two `TCPMSS --set-mss 1240` lines (`-i $NIC -o exit0` and `-i exit0 -o $NIC`), then
  `-i $NIC -o exit0 -j ACCEPT`, `-i exit0 -o $NIC -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT`,
  `-i $NIC -j REJECT --reject-with icmp-admin-prohibited`, `! -i exit0 -o $NIC -j REJECT`; FORWARD has
  `-j EXIT0` first. `EXIT0_NAT`: `-i $NIC ! -d $GW/32 -p udp --dport 53 -j DNAT --to-destination $GW:53`
  and the tcp twin. `EXIT0_MASQ`: `-s 10.a.b.0/24 -o exit0 -j MASQUERADE`. `ip6tables`:
  `-A FORWARD -i $NIC -j REJECT --reject-with icmp6-adm-prohibited`.
- sysctls `1 0 2 1`.
- Listeners: `127.0.0.1:10800` (NanoKVM-Server), `$GW:53` tcp and udp (NanoKVM-Server). Nothing on
  `0.0.0.0:10800`.
- Exactly one `udhcpd -S /etc/udhcpd.$NIC.conf`; the conf ends with `opt router 10.a.b.1` and `opt dns 10.a.b.1`.
- hev pidfile holds a live pid; the log is empty or has only startup lines (level warn).

UI: status card shows the NIC up with ifname/protocol and `10.a.b.1/24`, every `downstream` flag
green, tunnel **disconnected**, message empty (or `re-plug the USB cable …` if the rebind was refused;
then unplug/replug the consumer once and note it).

Pass: all of the above. Fail fast on any `S94exit` field `=0`: capture per the final section before
touching anything.

### 1.2 Commands are templated from the request

Exit browser: the Exit panel's Native tab shows three commands. Also fetch them raw via devtools:
Network → `/api/extensions/exit/0/commands` → response. Expected: `scheme` and `host` equal the
address bar; `fingerprint` is 64 lowercase hex on https and empty on http; `wstunnelVersion`
`10.7.1`; `wstunnelRepo` is `https://github.com/erebe/wstunnel`; `wstunnelLatest` has exactly three
entries, `windows` (powershell), `macos` (bash) and `linux` (bash), beside the three pinned ones in
`wstunnel`. The Windows command is
`irm -Headers @{Authorization='Bearer $TOKEN'} <scheme>://$KVM/exit/0/client.ps1 | iex`
(on https prefixed by `if (-not ('NanoKVMExitTrust' -as [type])) { Add-Type -TypeDefinition '<C#>' }; [NanoKVMExitTrust]::Install(); `,
a compiled trust-all callback for the fetch; a `{$true}` scriptblock there aborts the handshake under
Windows PowerShell 5.1 with `There is no Runspace available to run scripts in this thread`);
macOS and Linux are `curl -fsSL[k] -H 'Authorization: Bearer $TOKEN' <scheme>://$KVM/exit/0/client.sh | sh`
(`-k` only on https). The D28 warning text is visible above them; on http the panel says the token
travels in cleartext.

Fetch each script once by hand and check the templating took:

```sh
exit$ for f in client.sh client.py client.pl client.ps1; do echo "== $f"; curl -fsSk -H "Authorization: Bearer $TOKEN" -D - -o /tmp/$f https://$KVM/exit/0/$f | grep -iE '^(HTTP|content-type|cache-control)'; grep -c '__[A-Z_]*__' /tmp/$f; head -3 /tmp/$f; done
```

Expected: `HTTP/1.1 200`, `Content-Type: text/plain; charset=utf-8`, `Cache-Control: no-store`; the
placeholder count is `0` for every file; the first lines carry the language marker
(`nexit/1 python client` etc.). `grep -c "$TOKEN" /tmp/client.py` → `1` or more.

### 1.3 Windows PowerShell 5.1

Do `server/service/exit/clients/testdata/windows-checklist.md` §0 first (5.1, FullLanguage,
`Add-Type` compiles). Then paste the Windows command in a fresh Windows PowerShell window.

Expected output within ~3 s:

```
nexit: exit client for wss://$KVM/exit/0 (allowPrivate=False, pin=<64 hex>)
nexit: connected: streamWindow=131072 connWindow=4194304 maxStreams=256
```

(`ws://` and `pin=none` on http.) UI within 3 s: tunnel **connected**, peer `addr` = the laptop's
address as the NanoKVM sees it, `hostname` = the PC name, `os: windows`, `transport: native`;
`connectedAt` set; `upstream.reachable: true` with a `latencyMs` after ≤ 30 s (first probe).

```sh
kvm# grep -E 'exit0|exit: slot 0' /tmp/server.log | tail -5
kvm# netstat -tn | grep -E ':443 |:80 ' | grep ESTABLISHED
```

Expected: a session-established line, no error; one ESTABLISHED connection from the exit's address.

Ctrl+C the client: `nexit: stopping`; UI tunnel → disconnected within 3 s.

Pass: `connected:` line, UI shows the peer with `os: windows`, Ctrl+C is clean. On https also confirm
the pin: after Ctrl+C, `([Net.ServicePointManager]::ServerCertificateValidationCallback).Method.DeclaringType`
prints `NanoKVMExitTrust`, not `NexitPin` (the client restored the one-liner's fetch callback), and the
message `certificate was never presented for pinning` did **not** appear.

### 1.4 macOS

```sh
exit$ xcode-select -p; python3 -c 'import ssl, socket, selectors; print("py ok")'
exit$ <paste the macOS command>
```

Expected: `connected: streamWindow=131072 connWindow=4194304 maxStreams=256`; `pgrep -af python3`
shows the client (or `perl -e` if `xcode-select -p` failed: then also record that). Match on `python3`
rather than `python3 -c`: a Homebrew interpreter runs as
`…/Python.framework/Versions/3.14/Resources/Python.app/Contents/MacOS/Python -c`, which
`pgrep -fl 'python3 -c'` misses. The perl client says `allowPrivate=false` where the python one says
`allowPrivate=False`; both report the same slot. UI peer `os: darwin`,
`hostname` = `scutil --get LocalHostName`. On https: `client.sh` will first log curl `(60)` on the
verified fetch, then compare the served certificate's SHA-256 against the pin with `openssl s_client`
and refetch with `-k`; watch that it does **not** refetch when the pin is wrong (edit
`FINGERPRINT=` in a saved copy of `client.sh` to `000…0` and run it: expected `nexit: certificate
fingerprint mismatch … refusing`, exit 1, no connection on the UI).

Force the perl path once: `curl -fsSk -H "Authorization: Bearer $TOKEN" https://$KVM/exit/0/client.sh | env PATH=/usr/bin:/bin sh` with a PATH that has no python3 (make a dir of symlinks to `sh curl perl uname head grep sed tr openssl` if `/usr/bin` has python3). Expected: `perl -e` client, `connected:`.

Pass: both python3 and perl connect; wrong pin refuses.

### 1.5 Linux

```sh
exit$ python3 -c 'import ssl, socket, selectors; print("py ok")'; perl -MIO::Socket::SSL -e 'print "ssl ok\n"'
exit$ <paste the Linux command>
```

Expected as 1.4 with `os: linux`. Run once with python3 and once forced to perl (as above). On a
distro without `IO::Socket::SSL` the perl path on https must fail loudly, not silently connect
unverified: expected `perl` error naming `IO::Socket::SSL`, no session on the UI.

Pass: connects under both runtimes; every wrong-pin/no-pin variant refuses.

### 1.6 Watchdog re-asserts `ip_forward`

With the slot enabled and a client connected:

```sh
kvm# sysctl -w net.ipv4.ip_forward=0; sleep 35; sysctl -n net.ipv4.ip_forward; $S status 0 | head -1
```

Expected: `1` and `forward=1` (the 30 s converge ran `S94exit start 0`). UI `downstream.forward`
never showed red for more than one poll. Pass: `1`.

### 1.7 Egress, without the consumer

The SOCKS front door takes the same path the consumer's packets take once hev has turned them into
CONNECTs, so one curl on the NanoKVM shows whether the tunnel carries traffic before the consumer is
in the picture. It speaks only atyp 1 and 4, because hev only ever hands it addresses it read out of
an IP header; a name lands as atyp 3 and comes back REP 8, so `--socks5-hostname` cannot be used here
and names stay the forwarder's job (§11).

```sh
kvm# curl -s -m 25 --socks5 127.0.0.1:10800 https://1.1.1.1/cdn-cgi/trace | grep -E '^(ip|loc)='   # egress address
kvm# nslookup www.iana.org $GW | tail -6                                                           # the forwarder, same path
```

```
ip=203.0.113.9
loc=US
```

Expected: `ip=` is the **exit device's** public address, never the NanoKVM's own; `nslookup` answers
from `$GW`. With `--socks5-hostname` instead: `Can't complete SOCKS5 connection … (8)`, which is the
atyp rejection and not a tunnel fault. Pass: the exit's address, and a resolved name.

---

## 2. Mode B commands fetch the correct binary and establish the tunnel

### 2.1 Switch the slot to wstunnel

Exit browser: Exit panel → mode Segmented → **wstunnel**. (Sets `mode` via `POST /0/config`; the
manager restarts the listeners and converges the daemons.)

```sh
kvm# grep '"mode"' /etc/kvm/exit/0.json; grep MODE= /etc/kvm/exit/0/env
kvm# $S status 0 | grep -E 'wstunnel|hev'; cat /var/run/exit0-wstunnel.pid; pgrep -a wstunnel
kvm# cat /etc/kvm/exit/0/wstunnel-restrict.yml | sed "s/$TOKEN/<token>/"
kvm# netstat -lntp | grep -E ':10810|:10820|:10800'
kvm# tail -5 /tmp/exit0-wstunnel.log
```

Expected: `"mode": "wstunnel"`, `MODE=wstunnel`; `wstunnel=1 hev=1`; process
`wstunnel server --restrict-config /etc/kvm/exit/0/wstunnel-restrict.yml ws://127.0.0.1:10810`;
yaml has one restriction `name: exit0`, `!PathPrefix "^exit0$"`, `!Authorization "^Bearer <token>$"`,
`!ReverseTunnel` `protocol: [Socks5]` `port: [10820]` `cidr: [127.0.0.1/32]`; listeners `:10800`
(server) and `:10810` (wstunnel); **no** `:10820` yet. UI tunnel disconnected.

### 2.2 The commands

Panel → Wstunnel tab. Expected shape (from `commands.go`):

- Linux/macOS: `set -e; d=$(mktemp -d); cd "$d"; case "$(uname -m)" in x86_64) a=amd64 s=<sha>;; aarch64|arm64) a=arm64 s=<sha>;; *) …exit 1;; esac; curl -fsSLo wstunnel.tgz "https://github.com/erebe/wstunnel/releases/download/v10.7.1/wstunnel_10.7.1_<linux|darwin>_${a}.tar.gz"; echo "$s  wstunnel.tgz" | <sha256sum -c -|shasum -a 256 -c ->; tar -xzf wstunnel.tgz wstunnel; chmod +x wstunnel; exec ./wstunnel client -P exit/0 -H 'Authorization: Bearer $TOKEN' -R socks5://127.0.0.1:10820 wss://$KVM`
- Windows: `$v='10.7.1'; $a=…; $h=@{amd64='…';arm64='…'}[$a]; … Invoke-WebRequest … wstunnel_10.7.1_windows_$a.tar.gz …; if ((Get-FileHash $f -Algorithm SHA256).Hash.ToLower() -ne $h) { throw 'wstunnel checksum mismatch' }; tar -xzf $f -C $d; & (Join-Path $d 'wstunnel.exe') client -P exit/0 -H 'Authorization: Bearer $TOKEN' -R socks5://127.0.0.1:10820 wss://$KVM`
- `--tls-verify-certificate` appears **only** if the unit has a CA-signed certificate installed;
  with the stock self-signed one the note reads "wstunnel has no fingerprint pin; … unauthenticated".

Under the pinned command, every tab carries a latest command with its own description and the link
`https://github.com/erebe/wstunnel`. Expected shape:

- Linux, latest: `set -e; fail() { echo "could not fetch the latest wstunnel ($1); download it from https://github.com/erebe/wstunnel/releases" >&2; exit 1; }; d=$(mktemp -d); cd "$d"; case "$(uname -m)" in x86_64) a=amd64;; aarch64|arm64) a=arm64;; *) …exit 1;; esac; v=$(curl -fsSLo /dev/null -w '%{url_effective}' https://github.com/erebe/wstunnel/releases/latest) || fail resolve; v=${v##*/v}; f="wstunnel_${v}_linux_${a}.tar.gz"; curl -fsSLo wstunnel.tgz "https://github.com/erebe/wstunnel/releases/download/v$v/$f" || fail download; curl -fsSL "https://github.com/erebe/wstunnel/releases/download/v$v/checksums.txt" | grep " $f$" | sed 's/  .*/  wstunnel.tgz/' | sha256sum -c - || fail checksum; tar -xzf wstunnel.tgz wstunnel && chmod +x wstunnel || fail extract; exec ./wstunnel client -P exit/0 -H 'Authorization: Bearer $TOKEN' -R socks5://127.0.0.1:10820 wss://$KVM`
- macOS, latest: the Linux one with `_darwin_` in place of `_linux_` in the asset name and
  `shasum -a 256 -c -` in place of `sha256sum -c -` (macOS ships no `sha256sum`).
- Windows, latest: `try { $v=(Invoke-RestMethod -UseBasicParsing 'https://api.github.com/repos/erebe/wstunnel/releases/latest').tag_name.TrimStart('v'); $a=…; $n="wstunnel_${v}_windows_$a.tar.gz"; … Invoke-WebRequest … /releases/download/v$v/$n …; Invoke-WebRequest … /releases/download/v$v/checksums.txt …; $h=(Select-String -SimpleMatch -Pattern "  $n" -Path $s | Select-Object -First 1).Line; if (-not $h -or $h.Split(' ')[0] -ne (Get-FileHash $f -Algorithm SHA256).Hash) { throw 'wstunnel checksum mismatch' }; tar -xzf $f -C $d; & (Join-Path $d 'wstunnel.exe') client -P exit/0 -H 'Authorization: Bearer $TOKEN' -R socks5://127.0.0.1:10820 wss://$KVM } catch { throw "could not fetch the latest wstunnel: $_ Download it from https://github.com/erebe/wstunnel/releases" }`
- All three run whatever `https://github.com/erebe/wstunnel/releases/latest` resolves to (10.7.1 at the
  time of writing, the same as the pin) and verify the download against that tag's own
  `checksums.txt`, so a mismatch means the download was corrupted in transit; a tampered release
  would pass this check, and the pinned commands are the verified path. `--tls-verify-certificate`
  follows the pinned command of the same platform.

Independently confirm the pinned hashes against upstream before trusting the run:

```sh
exit$ curl -fsSL https://github.com/erebe/wstunnel/releases/download/v10.7.1/checksums.txt | grep -E 'linux_(amd64|arm64)|darwin_(amd64|arm64)|windows_(amd64|arm64)'
```

Expected: the six hashes equal the ones in the pasted commands (`fa842ed5…`, `99f9506d…`, `ac234c60…`,
`2c1f427f…`, `deb3c8b8…`, `1d642b29…`).

### 2.3 Run it, each platform

Linux and macOS:

```sh
exit$ <paste>
```

Expected stdout: `wstunnel.tgz: OK` from the checksum step, then wstunnel's own log:
`Starting wstunnel client v10.7.1`, `Connected to wss://$KVM` (or `Server listening on socks5://127.0.0.1:10820` echoed from the server side) and no `error`. `ps -o args= -p $(pgrep -n wstunnel)`
shows `./wstunnel client -P exit/0 …`; `./wstunnel --version` in `$d` prints `10.7.1`. The release
tarball stores the binary as `0644`, so the command sets the execute bit itself: `ls -l "$d"/wstunnel`
shows `-rwxr-xr-x`.

Windows:

```powershell
exit> <paste>
```

Expected: no `wstunnel checksum mismatch` throw; `tar` is the Windows 10 1803+ built-in; the
client's log as above. If the run is on an ARM64 PC the `$a` branch must pick `arm64`.

Microsoft Defender quarantines the wstunnel binary on a stock Windows 11 exit, the pinned release
and the latest alike, as `Trojan:Win32/Bearfoos.B!ml`, which `Get-MpThreatDetection` names. A fresh
start is refused with `Operation did not complete successfully because the file contains a virus or
potentially unwanted software`, and a client already running dies about 90 s in, once the cloud
lookup lands, with nothing in its own log; at the consumer the bound requests come back as
`curl: (35) Recv failure: Connection was reset` in about 200 ms. Under a Defender path exclusion on
the extraction directory the same command connects and carries traffic. The panel's Windows
wstunnel tab carries this warning.

Still current, re-confirmed on signature `1.459.270.0` with engine `1.1.26080.3`: the download and
the checksum pass, `tar` extracts a 10344448-byte `wstunnel.exe`, `--version` prints `wstunnel-cli
10.7.1`, and about 110 s later the file is simply gone, with one `Get-MpThreatDetection` row,
`ThreatID 2147731849`, `SeverityID 5`, `ActionSuccess True`, naming the extracted path. With
`Add-MpPreference -ExclusionPath` on that directory the same sequence survives 115 s with no
detection at all, so the exclusion is the whole difference.

The latest commands, each platform:

```sh
exit$ <paste the latest command>
```

Expected on Linux and macOS: `wstunnel.tgz: OK`, then `Starting wstunnel client v<ver>` where `<ver>` is the
tag `https://github.com/erebe/wstunnel/releases/latest` shows in a browser. `<ver>` has already moved a
major above the pinned `10.7.1`, and the restriction the kvm side writes does not depend on it: a v11
client connects, opens the reverse listener and carries traffic against the same
`wstunnel-restrict.yml`. Windows prints the same
without the `OK` line (`Get-FileHash` does the comparison and a mismatch throws `wstunnel checksum
mismatch`). Break one run on purpose (cut the exit's network before pasting, or change `_linux_` /
`_darwin_` / `_windows_` in a saved copy to `_nope_`): Linux and macOS print `could not fetch the
latest wstunnel (download); download it from https://github.com/erebe/wstunnel/releases` and exit 1
(the step in parentheses is one of `resolve`, `download`, `checksum`, `extract`), Windows throws
`could not fetch the latest wstunnel: … Download it from https://github.com/erebe/wstunnel/releases`;
no wstunnel process starts in any case.

On the NanoKVM, within 3 s of the client connecting:

```sh
kvm# netstat -lntp | grep ':10820'; tail -3 /tmp/exit0-wstunnel.log
kvm# grep -E 'exit0.*(wstunnel|peer|reverse)' /tmp/server.log | tail -3
```

Expected: `127.0.0.1:10820` LISTEN owned by `wstunnel`; UI tunnel **connected** (needs the Go proxy's
500 ms reverse-listener probe to have succeeded once), peer `transport: wstunnel`, peer `addr` = the
exit's address (hostname/os are empty in Mode B). `upstream.reachable: true` within 30 s.

```sh
exit$ curl -sS -o /dev/null -w '%{http_code}\n' --max-time 10 http://example.com/   # the exit's own internet, sanity
```

Then §3 step 3.3 through this mode as well. Pass: all three platforms verify the pinned hash and run
10.7.1, the two latest commands run the version the releases page shows, the reverse listener appears,
UI connected, the consumer reaches the internet (§3) in Mode B.

### 2.4 Restrictions hold

```sh
exit$ ./wstunnel client -P exit/0 -H "Authorization: Bearer $TOKEN" -L tcp://127.0.0.1:2222:127.0.0.1:22 wss://$KVM
exit$ ./wstunnel client -P exit/0 -H "Authorization: Bearer $TOKEN" -R tcp://0.0.0.0:2222:127.0.0.1:22 wss://$KVM
exit$ curl -sk -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $TOKEN" https://$KVM/exit/0/exit0/events      # not an upgrade
exit$ curl -sk -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $TOKEN" https://$KVM/exit/0/native           # Mode A route in Mode B
```

Expected: the forward tunnel and the non-loopback reverse tunnel are refused (wstunnel client logs an
upgrade error / the server log shows the restriction denying it; `netstat -lntp | grep 2222` on the
NanoKVM stays empty); both curls print `404`. Pass: nothing but the reverse SOCKS on 10820 is ever
accepted.

Switch back to **native** for the rest of the checklist (or repeat §3–§11 in both modes if time
allows; at minimum do §3, §4, §5, §10 in Mode B once). After switching back:
`$S status 0 | grep wstunnel` → `wstunnel=1`, `pgrep wstunnel` → nothing, `:10810` and `:10820` gone.

---

## 3. Consumer gets a DHCP lease, resolves DNS, reaches the internet through the tunnel

Precondition: slot enabled (§1.1), a Mode A client connected (§1.3–1.5), consumer's other uplinks
off.

### 3.1 The lease

```powershell
win> ipconfig /all
win> Get-NetAdapter | ? Status -eq Up | ft Name,InterfaceDescription,LinkSpeed
win> Get-NetIPConfiguration -Detailed | ? { $_.IPv4Address.IPAddress -like '10.*' }
win> Get-NetRoute -AddressFamily IPv4 -DestinationPrefix 0.0.0.0/0 | ft ifIndex,NextHop,RouteMetric,InterfaceMetric
win> Get-DnsClientServerAddress -AddressFamily IPv4 | ? ServerAddresses
```

```sh
mac$ IF=$(route -n get default | awk '/interface:/{print $2}'); echo $IF; ipconfig getpacket $IF
mac$ ifconfig $IF | grep 'inet '; netstat -rn -f inet | head -5; scutil --dns | grep -A3 'resolver #1'
```

```sh
lin$ IF=$(ip -4 route show default | awk '{print $5}' | head -1); echo $IF; ip -4 addr show $IF; ip route
lin$ resolvectl status $IF 2>/dev/null || cat /etc/resolv.conf
lin$ nmcli -f DHCP4 dev show $IF 2>/dev/null | grep -E 'routers|domain_name_servers|ip_address|expiry|dhcp_lease_time'
```

Expected: the NanoKVM's interface (Windows description "USB Ethernet/RNDIS Gadget" or "CDC-NCM";
macOS `enX` "USB 10/100/1000 LAN"-style; Linux `usb0`/`enx…`) has `10.a.b.1xx/24` (`.100`–`.200`),
**Default Gateway `10.a.b.1`**, **DHCP Server `10.a.b.1`**, **DNS Server `10.a.b.1`**, lease
duration 864000 s (Windows shows "Lease Expires" ten days out; `ipconfig getpacket` shows
`lease_time (uint32): 0xd2f00`; nmcli `dhcp_lease_time = 864000`). The default route's next hop is
`10.a.b.1` on that interface. `ipconfig getpacket` shows `router (ip_mult): {10.a.b.1}` and
`domain_name_server (ip_mult): {10.a.b.1}`.

```sh
kvm# cat /var/lib/misc/udhcpd.$NIC.leases | xxd | head -3; iptables -t nat -L EXIT0_NAT -v -n -x | head -4
```

Pass: gateway and DNS present and equal to `$GW`; exactly one lease.

### 3.2 DNS

```powershell
win> nslookup www.iana.org
win> nslookup -type=AAAA www.iana.org
win> Resolve-DnsName www.iana.org -Server 10.a.b.1 | ft Name,Type,IPAddress
```

```sh
mac$ dig example.com +noall +answer +comments | head; dig AAAA example.com +noall +comments +answer
lin$ dig example.com +noall +answer +comments | head; dig AAAA example.com +noall +comments +answer; dig +tcp example.com +short
```

Expected: A answers with `SERVER: 10.a.b.1#53`, `status: NOERROR`; AAAA is `status: NOERROR` with
**zero** answers (D22 filter); `+tcp` works too. Windows `nslookup -type=AAAA` prints the SOA/"No
records" style output, not a timeout.

```sh
kvm# $S status 0 | grep dns_redirected      # still 0: the consumer used the lease's resolver
```

UI `dns.queries` increments by the number of lookups, `dns.failures` 0.

Pass: A resolves via `$GW`, AAAA empty NOERROR, no timeouts.

### 3.3 Internet

```powershell
win> curl.exe -4 -sS -o NUL -w "%{http_code} %{remote_ip} %{time_total}s`n" https://www.iana.org/
win> curl.exe -4 -sS https://api.ipify.org; echo
win> Test-NetConnection -ComputerName 1.1.1.1 -Port 443 | ft TcpTestSucceeded,RemoteAddress
win> tracert -d -4 -h 4 1.1.1.1
```

```sh
mac$ curl -4 -sS -o /dev/null -w '%{http_code} %{remote_ip} %{time_total}s\n' https://example.com/; curl -4 -sS https://api.ipify.org; echo
mac$ nc -vz -w 5 1.1.1.1 443; traceroute -n -m 4 1.1.1.1
lin$ curl -4 -sS -o /dev/null -w '%{http_code} %{remote_ip} %{time_total}s\n' https://example.com/; curl -4 -sS https://api.ipify.org; echo
lin$ nc -vz -w 5 1.1.1.1 443; traceroute -n -m 4 1.1.1.1
```

```sh
exit$ curl -4 -sS https://api.ipify.org; echo     # the exit's own public address
```

Expected: `200`, `time_total` under ~2 s; `api.ipify.org` on the consumer prints **the exit
device's** public IP (same as the exit's own curl), not the NanoKVM's LAN's; TCP 443 to 1.1.1.1
succeeds. `tracert`/`traceroute` shows hop 1 `10.a.b.1` and then `*` for the rest: ICMP is not
carried by tun2socks, so this is the expected shape, not a failure (`ping 8.8.8.8` also fails; note it).

```sh
kvm# iptables -L EXIT0 -v -n -x | grep -E 'ACCEPT|TCPMSS'; iptables -t nat -L EXIT0_MASQ -v -n -x | tail -1
kvm# ip -s link show exit0; (conntrack -L 2>/dev/null || cat /proc/net/nf_conntrack) | grep -c "src=10\.[0-9]*\.[0-9]*\.1[0-9][0-9] "
kvm# tail -3 /tmp/exit0-hev.log; grep -c . /tmp/exit0-hev.log
```

Expected: ACCEPT and TCPMSS packet counters non-zero and rising, MASQUERADE counter non-zero;
`exit0` RX/TX bytes rising, `errors 0 dropped 0`; conntrack entries for the consumer's `10.a.b.1xx`;
the hev log grows by nothing (warn level) or by benign lines. UI `bytes.up/down` rising,
`upstream.reachable: true`.

On the nf_tables-backed image the per-rule counters in `iptables -L EXIT0 -v -n -x` read 0 (the chain
is replaced every 30 s and this build does not maintain them), so use conntrack, `ip -s link show exit0`
and the status byte counters as the pass criterion instead.

Pass: 200 with the exit's public IP; counters on `EXIT0`/`EXIT0_MASQ`/`exit0` move.

### 3.4 Browser on the consumer

Open `https://www.iana.org`, `https://www.wikipedia.org`, a YouTube video for 60 s, and the NanoKVM UI
at `https://10.a.b.1`. Expected: all load; the UI loads without touching the tunnel (`local` route:
`ip route get 10.a.b.1 from 10.a.b.1xx iif $NIC` on the NanoKVM prints `local`). Pass: no page stalls.

### 3.5 Policy: the exit's LAN is refused

```powershell
win> curl.exe -sS -m 5 -o NUL -w "%{http_code} %{exitcode}`n" http://192.168.1.1/    # any RFC1918 address the exit could reach
```

```sh
mac$ curl -sS -m 5 -o /dev/null -w '%{http_code} %{exitcode}\n' http://192.168.1.1/; curl -sS -m 5 http://169.254.169.254/
```

Expected: immediate failure (curl exit 7 "Connection refused" / "No route to host" within
milliseconds, from the `EXIT0` REJECT with `icmp-admin-prohibited`), not a 5 s timeout; the
`-d 192.168.0.0/16 -j REJECT` counter in `iptables -L EXIT0 -v -n -x` increments. Then Advanced →
allowPrivate on: `REJECT4` in `/etc/kvm/exit/0/env` shrinks to the five always-deny prefixes, the
chain loses the four private REJECTs, and the same curl now reaches (or times out at) the exit's
router; the front door and the client would each refuse if only the chain were relaxed, so a reach
here proves all three layers moved together. Turn allowPrivate back off.

A Windows consumer keeps retransmitting the SYN through the ICMP admin-prohibited replies and reports
curl exit 28 at its deadline, while a usb0 capture (`tcpdump -ni usb0 -l -c 6 'icmp or tcp port 80'`)
shows the kvm's ICMP within a millisecond of each SYN; on Windows the pass criterion is the capture,
not the exit code. On the nf_tables-backed image the REJECT counter reads 0 like every other per-rule
counter in `iptables -L EXIT0 -v -n -x` (the chain is replaced every 30 s and this build does not
maintain them), so the evidence is conntrack, `ip -s link show exit0` or the capture, never the counter.

The list covers forwarded traffic only. `EXIT0` hangs off `FORWARD`, so a packet the consumer addresses
to one of the NanoKVM's own addresses lands in `INPUT` instead and never reaches the chain, and `INPUT`
is policy ACCEPT with two eth0 rules. Both `$GW` and the tun's `198.18.0.1` therefore answer on every
port the unit binds with a wildcard: `curl http://198.18.0.1/` returns the unit's own `307` in about
5 ms and TCP 22 opens on both, while the routed `198.18.0.2` beside it is refused as the list says.
The consumer already reaches the unit at `$GW`, so this is the reach of the REJECT list and not a
second way in, but a slot that is meant to hide the unit from the consumer needs an `INPUT` rule of
its own.

Pass: fast refusal by default; `198.18.0.0/15` except `198.18.0.1`, `127/8` and `169.254/16` refused
in both settings.

### 3.6 UDP through the tunnel

```sh
mac$ dig @1.1.1.1 example.com +notcp +short          # DNAT'd to $GW:53 anyway, see §11; so use a non-53 UDP service:
mac$ python3 -c "import socket,struct,time;s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);s.settimeout(5);s.sendto(b'\x1b'+47*b'\0',('time.cloudflare.com',123));d,a=s.recvfrom(48);print('ntp reply from',a,struct.unpack('!I',d[40:44])[0]-2208988800)"
```

```powershell
win> w32tm /stripchart /computer:time.cloudflare.com /samples:2 /dataonly
```

Expected: an NTP reply (a Unix timestamp near now). This is SOCKS UDP ASSOCIATE → nexit UDP stream
→ the exit's UDP socket. Pass: reply within 5 s. In Mode B this exercises wstunnel's UDP server
(Risk 3): repeat there.

### 3.7 Many streams, and the `maxStreams` limit

```sh
mac$ curl -sS --parallel --parallel-immediate --parallel-max 50 -o /dev/null -w '%{http_code}\n' $(for i in $(seq 50); do echo "https://example.com/?i=$i"; done) | sort | uniq -c
```

Expected: `50 200`. On a Windows consumer count curl's own exit code instead: `--parallel` loses some
of its `-w` lines when stdout is redirected, so the tally reads 40 to 45 of 50 across runs while every
transfer in fact succeeded. `curl.exe -s -S --parallel ... 1>codes.txt 2>errs.txt` then `$LASTEXITCODE`
of `0` with an empty `errs.txt` is the pass, and the `200`s that do land confirm the shape.

Then hold 256+ idle connections open:

```sh
mac$ python3 -c "
import socket,time
s=[]
for i in range(300):
    try:
        c=socket.create_connection(('1.1.1.1',443),timeout=5); s.append(c)
    except Exception as e: print(i,e); break
print('open',len(s)); time.sleep(60)"
```

This does not reach the cap, and the reason is worth knowing before reading the numbers. hev completes
the consumer's TCP handshake in its own stack and opens an upstream session only when data first
flows, then reaps that session once the connection goes quiet while leaving the consumer's socket up.
So 300 idle connections cost 6 front-door sessions, and every `connect` succeeds however many you
open. Driving a TLS handshake on all 300 as they are opened does not reach it either: all 300
complete with no failure, the front door peaks near 126 concurrent sessions because the early ones
are already being reaped, and it falls back to 8 while the 300 sockets sit open.

Reaching `max-session-count: 256`/`maxStreams=256` therefore needs 256+ connections each carrying
traffic continuously, not merely open. Record what you see: `ss -tn state established '( sport = :10800 )'`
on the kvm is the live session count, and the consumer's own socket count is not it.

Expected as written here: 300 connects succeed, 300 TLS handshakes succeed, front door peaks around
126 and settles to single digits when the connections idle, and nothing in `/tmp/exit0-hev.log`.
While they are open, take the measurements in 3.9.

### 3.8 Idle keepalive

Leave the tunnel idle for 5 minutes (no consumer traffic). Expected: UI stays **connected** the
whole time (`uptimeSeconds` climbs past 300, no `connectedAt` reset); the client prints nothing
about reconnecting; then §3.3's curl works first try. Pass: no reconnect in 5 min idle.

### 3.9 RSS at idle and under 256 sessions

Run at idle (after 3.8) and again during 3.7's hold:

```sh
kvm# for p in "hev $(cat /var/run/exit0-hev.pid 2>/dev/null)" "wstunnel $(cat /var/run/exit0-wstunnel.pid 2>/dev/null)" "server $(pidof NanoKVM-Server)"; do set -- $p; [ -n "$2" ] && printf '%-9s pid=%-6s %s %s\n' "$1" "$2" "$(grep VmRSS /proc/$2/status | tr -s ' ')" "$(grep VmSize /proc/$2/status | tr -s ' ')"; done; free -m | head -2
```

Expected: hev RSS a few MB idle, under ~16 MB at 256 sessions and always below the 65536 KiB
address-space cap S94exit sets (`ulimit -v`; if `VmSize` approaches 65536 kB hev will start failing
allocations: watch `/tmp/exit0-hev.log`); server RSS grows by at most ~256 × (128 KiB window + 16 KiB
frame) ≈ 36 MB worst case over 0.11's baseline and should come back down after the sockets close;
wstunnel (Mode B) similar to the tunnel page's numbers. Record all six numbers. Pass: no process
killed, `free` shows no swap thrash (`Swap: used` steady), the D24 ≈ 24 MB target is within 2× or the
discrepancy is filed.

Measured on this unit with 300 consumer connections and 300 completed TLS handshakes: hev RSS 2248 kB
at idle, 6284 kB at the peak and back to 3500 kB once they went quiet, VmSize 12916 kB against the
65536 kB `ulimit -v`; NanoKVM-Server RSS 52256 kB; `free -m` steady at 157 total with no swap
movement; `/tmp/exit0-hev.log` empty throughout.

---

## 4. Exit device disconnect

Two variants: graceful (Ctrl+C) here, ungraceful in §9.

### 4.1 Disconnect

```sh
exit$ <Ctrl+C in the client window>     # Mode A prints "nexit: stopping"
```

or from the UI: Exit panel → **Disconnect** (`POST /0/disconnect`, close reason `disconnected by operator`).

Expected within one UI poll (3 s): tunnel **disconnected**, `lastConnectedAt` set to now, `peer` cleared,
`upstream` stops updating. `grep -E 'exit0.*(closed|session ended)' /tmp/server.log | tail -1`.

On a Windows exit in Mode B, `wstunnel.exe` outlives the PowerShell host that launched it, so the
teardown is `Stop-Process -Name wstunnel`: closing the window or stopping the wrapper leaves the
tunnel up.

### 4.2 Consumer NIC stays up and the lease holds

```powershell
win> Get-NetAdapter | ? Status -eq Up | ft Name,Status; ipconfig | Select-String -Context 0,6 'Ethernet adapter'
win> Get-NetRoute -DestinationPrefix 0.0.0.0/0 | ft NextHop
```

```sh
mac$ ifconfig $IF | grep -E 'status|inet '; route -n get default | grep gateway
lin$ ip -4 addr show $IF | grep inet; ip route show default; ip link show $IF | grep -o 'state [A-Z]*'
```

Expected: identical to §3.1: same address, same gateway, link up, no "Media disconnected". Nothing on
the consumer noticed anything at L2/L3.

```sh
kvm# $S status 0; echo rc=$?; ip rule | grep -E '^100[01]:'; ip route show table 100; cat /sys/class/net/exit0/carrier
```

Expected: unchanged from §1.1, `rc=0`, carrier `1`: exit connect/disconnect touches none of the
downstream (D8).

### 4.3 Internet stops, fast and closed

```powershell
win> curl.exe -4 -sS -m 10 -o NUL -w "%{http_code} %{exitcode} %{time_total}s`n" https://www.iana.org/
win> nslookup www.iana.org
win> Test-NetConnection 1.1.1.1 -Port 443 | ft TcpTestSucceeded
```

```sh
mac$ time curl -4 -sS -m 10 -o /dev/null https://example.com/; dig example.com +noall +comments | grep status
lin$ time curl -4 -sS -m 10 -o /dev/null https://example.com/; dig example.com +noall +comments | grep status
```

Expected: **DNS `status: SERVFAIL`** (Windows: "Server failed"), returned immediately (< 100 ms), not a
timeout; a curl to a literal IP (`curl -m 10 https://1.1.1.1/`) fails within ~1 s with connection
refused/reset, never the full 10 s (front door answers `0x03` in under a millisecond, hev RSTs).
The exit code follows where the RST lands: `7` or `56` when the connect itself is refused, and `35`
`Recv failure: Connection was reset` when it arrives during the TLS handshake, which is what a
Windows consumer reports (observed: `exit=35` at 216 ms). That is the same string §2.3 records for a
Defender kill, so read it together with the tunnel state: here the panel says disconnected and every
destination fails, while under Defender the tunnel is up and the client's own process is gone.
`Test-NetConnection` `False` quickly.

```sh
kvm# tcpdump -ni eth0 -c 20 -w /tmp/leak.pcap 'not port 22 and (src net 10.0.0.0/8 or dst net 10.0.0.0/8) and not host '"$GW"'' & sleep 1
mac$ curl -4 -sS -m 5 http://93.184.216.34/ ; curl -4 -sS -m 5 http://1.1.1.1/     # generate a few packets, then:
kvm# sleep 6; kill %1 2>/dev/null; tcpdump -nr /tmp/leak.pcap 2>/dev/null | head
kvm# iptables -L EXIT0 -v -n -x | grep -E 'REJECT.*-i '"$NIC"'\s*\*' ; ip -s link show eth0 | sed -n '3,4p'
```

Expected: **zero** packets from the consumer's `10.a.b.1xx` on eth0 (the gadget subnet must never
appear on the LAN, masqueraded or not). Without tcpdump: the consumer's connections show in
conntrack only with `src=10.a.b.1xx dst=<dest>` and **no** `src=<eth0 address>` translation, and eth0's
TX packet counter does not move in step with the consumer's attempts. On the nf_tables-backed image
the REJECT counter in `iptables -L EXIT0 -v -n -x` reads 0 (per-rule counters are not maintained on
this build), so conntrack, `ip -s link show exit0` and the status byte counters carry the check.
Pass: SERVFAIL + fast refusal + zero leak.

### 4.4 NanoKVM reachability persists

```powershell
win> ping -n 2 10.a.b.1; curl.exe -sk -o NUL -w "%{http_code}`n" https://10.a.b.1/; ssh root@10.a.b.1 uptime
```

```sh
mac$ ping -c 2 $GW; curl -sk -o /dev/null -w '%{http_code}\n' https://$GW/; ssh root@$GW uptime
```

Expected: ping replies, `200`, ssh works. The browser on the consumer can log in to the UI and see
the stream (the consumer already reaches UI/SSH/VNC on the gadget address; the feature does not
change that, D28). Pass: all three.

---

## 5. Exit device reconnect

### 5.1 Manual reconnect (after a Ctrl+C)

```sh
exit$ <paste the command again>
```

Expected: `connected:` within ~3 s; UI **connected**, `connectedAt` new, `lastConnectedAt` from §4;
`previousPeer` unchanged (it records the last change of peer, not a live replacement). Then, with no action on the consumer:

```powershell
win> curl.exe -4 -sS -o NUL -w "%{http_code}`n" https://www.iana.org/; nslookup www.iana.org
```

```sh
mac$ curl -4 -sS -o /dev/null -w '%{http_code}\n' https://example.com/; dig example.com +short
```

Expected: `200` first try, DNS answers, no `ipconfig /renew`, no cable pull. Browser tabs left on
"can't reach" reload fine.

### 5.2 Automatic reconnect (the kvm side drops)

With the client running: UI → **Disconnect**. Expected on the exit: `session ended: websocket closed
by kvm (1000 disconnected by operator)` then `reconnecting in 1s` → `connected:` (backoff resets to 1
after a session that reached WELCOME). UI connected again within ~5 s. Consumer curl works again.

Then restart the server while the client stays up:

```sh
kvm# /etc/init.d/S95nanokvm restart > /root/server.out 2>&1 < /dev/null    # or kill and re-run the foreground command from Conventions
```

Expected on the exit: `session ended … / reconnecting in 1s / 2s / 4s …` until the server is back
(≈10–20 s), then `connected:`. Expected on the NanoKVM after the server is up: `$S status 0` all `1`
(Init re-ran `S94exit start 0` and rebound `:10800` and `$GW:53`), `ps w | grep '[u]dhcpd'` still exactly one.
Consumer: internet resumes untouched. Pass: resumes within 60 s without consumer action; the client
never exceeds 30 s between attempts.

---

## 6. NanoKVM reboot

Slot enabled, client running on the exit (leave it running: it reconnects on its own), consumer
plugged in with `ipconfig`/`ifconfig` visible.

### 6.1 Reboot and record the timeline

```sh
kvm# date; reboot
```

Immediately reconnect ssh over eth0 and run the 0.7 timeline loop (it now also shows `exit0=` and
`hev=`). On the consumer run a watch in parallel:

```powershell
win> while ($true) { "$(Get-Date -f HH:mm:ss) $((Get-NetAdapter | ? InterfaceDescription -match 'RNDIS|NCM|Gadget').Status) $((Get-NetIPAddress -AddressFamily IPv4 | ? IPAddress -like '10.*').IPAddress) $((Get-NetRoute -DestinationPrefix 0.0.0.0/0 -ea 0).NextHop)"; Start-Sleep 2 }
```

```sh
mac$ while true; do echo "$(date +%T) $(ifconfig $IF 2>/dev/null | awk '/status/{print $2} /inet /{print $2}' | tr '\n' ' ') gw=$(route -n get default 2>/dev/null | awk '/gateway/{print $2}')"; sleep 2; done
lin$ while true; do echo "$(date +%T) $(ip -br -4 addr show $IF 2>/dev/null) gw=$(ip -4 route show default | awk '{print $3}')"; sleep 2; done
```

Expected order on the NanoKVM timeline (uptime seconds in brackets are what you fill in):

1. `[  ]` `exit0=[0] hev=[pid]` **before** `server=[pid]`: `S94exit start` at boot brought up sysctls,
   tun, table 100 and hev with no NIC (`(unnamed net_device)`), and logged
   `slot 0: no gadget NIC registered yet; rules and chains wait for the bind` (check
   `dmesg | grep -i S94exit` or the boot console; rcS output goes to the console).
2. `[  ]` `server=[pid]`.
3. `[  ]` `udc=[4340000.usb]`, `ifname=[usbN]`, then within ~2 s `addr=[10.a.b.1/24]` and one
   `udhcpd` pid (post-attach hook: `S30rndis start`, `S94exit start 0`).
4. `[  ]` consumer watch: adapter Up, address `10.a.b.1xx`, gateway `10.a.b.1` (a fresh lease; the
   address may be the same as before).

Then:

```sh
kvm# $S status 0; echo rc=$?; ip rule | grep -E '^100[01]:'; cat /etc/udhcpd.$NIC.conf | tail -2; ps w | grep '[u]dhcpd' | wc -l; cat /etc/kvm/exit/gadget.route
kvm# netstat -lntp | grep -E ':10800|:53 '; iptables -t nat -S EXIT0_NAT | grep DNAT | head -1
kvm# sysctl -n net.ipv4.ip_forward      # S98tailscaled ran after S94exit and zeroed it; the server's Init/watchdog must have re-set it
```

Expected: all `1`, `rc=0`; the udhcpd conf still has `opt router`/`opt dns` (marker survived);
exactly `1` udhcpd; `:10800` and `$GW:53` listening; the DNAT rule names the current `$GW`;
`ip_forward` `1`.

### 6.2 The tunnel server waits, then the exit resumes internet

Consumer, before the exit reconnects (the client's backoff can take up to 30 s):

```sh
mac$ dig example.com +noall +comments | grep status; curl -4 -sS -m 5 -o /dev/null -w '%{exitcode}\n' https://1.1.1.1/
```

Expected: `SERVFAIL` and a fast refusal (§4.3 behaviour), never a leak. Exit window: `reconnecting in
…` then `connected:`. Consumer:

```sh
mac$ curl -4 -sS -o /dev/null -w '%{http_code}\n' https://example.com/
```

Expected: `200`, no consumer-side action. Record: power-off → NIC up on the consumer (s), and → first
successful consumer curl (s). Pass: downstream converged without the UI being opened, one udhcpd,
lease with gateway, internet back once the exit reconnected.

The kvm half converges on its own and needs nothing: after an unplanned power-cycle reboot this unit
came back with rules `1000`/`1001`, table 100, `exit0` up at carrier 1, `127.0.0.1:10800` and `$GW:53`
bound, one udhcpd, every `S94exit` field 1, `hev_spawns` 1 and an empty hev log, and the exit client
reconnected by itself on its documented backoff. Cumulative `dns_redirected` resets to 0, because
`stop` clears it.

The consumer half does not follow automatically. The gadget rebinds during boot (`dmesg` shows
`configfs-gadget gadget: uvc: uvc_function_bind()` around 18 s) and `/sys/kernel/config/usb_gadget/g0/UDC`
holds the controller, yet `cat /sys/class/udc/*/state` can read `not attached` with `usb0` `NO-CARRIER`
and an empty lease file, which is the host never re-enumerating rather than anything the slot did.
Check that state before reading any consumer result after a reboot; it is fixed from the consumer,
by rescanning or by cycling the composite device there, not from the kvm.

### 6.3 Boot with the exit already gone

Stop the exit client, reboot the NanoKVM again. Expected: identical downstream state; consumer
gets the lease with gateway/DNS and sees SERVFAIL/refusal (no leak) until an exit connects. UI
tunnel disconnected, `lastConnectedAt` restored from `/etc/kvm/exit/0.state.json`.

---

## 7. Token rejection: invalid token refused, no information leak

Run this whole section **from the NanoKVM against itself**, so the failures lock source `127.0.0.1`
and never the exit laptop (a source that fails 27 times is locked for the 15 min cap, and even the
correct token is refused while locked). Find the server's port and set the base:

```sh
kvm# netstat -lntp | grep NanoKVM-Server        # e.g. 0.0.0.0:443 or 0.0.0.0:80
kvm# H2=https://127.0.0.1                        # or http://127.0.0.1:<port>
```

Order matters: 7.1 (reference), 7.2 (rate limiting, five failures), 7.3 (identity sweep, locks
`127.0.0.1` for up to 15 min: harmless), 7.4 (logs, no requests).

### 7.1 Reference 404

```sh
kvm# curl -sk -i $H2/this-route-does-not-exist-9f3a | sed '/^[Dd]ate:/d' | tee /tmp/ref404.txt
```

Expected: `HTTP/1.1 404 Not Found`, `Content-Type: text/plain`, `Content-Length: 18`, body
`404 page not found`. This route is not rate limited, so it is a stable reference for the rest.

### 7.2 Rate limiting

```sh
kvm# for i in 1 2 3 4 5; do printf '%s wrong attempt %d: ' "$(date +%T)" $i; curl -sk -o /dev/null -w '%{http_code}\n' -H 'Authorization: Bearer wrongtok1' $H2/exit/0/client.sh; done
kvm# printf 'correct while locked: '; curl -sk -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $TOKEN" $H2/exit/0/client.sh
kvm# sleep 3; printf 'correct after 3 s: '; curl -sk -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $TOKEN" $H2/exit/0/client.sh
```

Expected: five `404`s; failures 4 and 5 impose lockouts of 1 s and 2 s (failure n > 3 locks for
`1 s << (n-4)`: 1, 2, 4, 8, 16 … s, cap 15 min; entries are forgotten after 1 h idle); the correct
token **while locked** → `404`; after `sleep 3` → `200`. Pass: correct-while-locked is 404,
correct-after is 200.

### 7.3 Every rejection is byte-identical to the reference

```sh
kvm# for u in "$H2/exit/0/client.sh" "$H2/exit/0/native" "$H2/exit/0" "$H2/exit/7/client.sh" "$H2/exit/0/client.py"; do
  for a in "" "Authorization: Bearer wrongtok1" "Authorization: Bearer ${TOKEN}x" "Authorization: Basic Zm9v" "Authorization: bearer $(echo $TOKEN | tr a-z A-Z)"; do
    curl -sk -i ${a:+-H "$a"} "$u" | sed '/^[Dd]ate:/d' > /tmp/got.txt
    if diff -q /tmp/ref404.txt /tmp/got.txt >/dev/null; then echo "IDENTICAL  $u  [$a]"; else echo "DIFFERS    $u  [$a]"; diff /tmp/ref404.txt /tmp/got.txt; fi
    sleep 0.3
  done
done
```

Expected: every one of the 25 lines `IDENTICAL`: no auth, wrong token, correct token with a trailing
character, wrong scheme, upper-cased token (the `bearer` scheme is case-insensitive, the token
compare is exact), bare `/exit/0`, unknown slot `7`. No `WWW-Authenticate`, no `Set-Cookie`, no
JSON envelope, no different `Content-Length`. From about the 4th request on, the source is also
locked, so the sweep covers the "locked source" 404 as well; an unknown slot costs the same time as
a wrong token (the gate compares against an 8×`x` placeholder).

WebSocket upgrade specifically, with and without a header:

```sh
kvm# curl -sk -i -H "Authorization: Bearer wrongtok1" -H 'Connection: Upgrade' -H 'Upgrade: websocket' -H 'Sec-WebSocket-Version: 13' -H 'Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==' $H2/exit/0/native | sed '/^[Dd]ate:/d' | diff /tmp/ref404.txt - && echo WS-IDENTICAL
kvm# curl -sk -i -H 'Connection: Upgrade' -H 'Upgrade: websocket' -H 'Sec-WebSocket-Version: 13' -H 'Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==' $H2/exit/0/native | sed '/^[Dd]ate:/d' | diff /tmp/ref404.txt - && echo WS-NOAUTH-IDENTICAL
```

Expected: `WS-IDENTICAL` and `WS-NOAUTH-IDENTICAL`; no `101`, no `Sec-WebSocket-Accept`.

Two more, later: in §2.4 (Mode A route while in Mode B → 404) and after §12's disable
(`curl -sk -i -H "Authorization: Bearer $TOKEN" $H2/exit/0/client.sh` with the slot off → IDENTICAL,
correct token notwithstanding). Timing: `curl -sk -o /dev/null -w '%{time_total}\n'` for the
unknown-slot and wrong-token cases should be within a millisecond or two of each other. Over https on
this unit each request spends 140 to 190 ms in the TLS handshake and the two cases interleave with no
consistent ordering across samples, so compare several rounds and look for a consistent gap rather
than a single pair: the handshake noise is an order of magnitude larger than the compare.

### 7.4 No leak into logs

```sh
kvm# grep -c "$TOKEN" /tmp/server.log /tmp/exit0-hev.log /tmp/exit0-wstunnel.log 2>/dev/null
kvm# grep -iE 'wrongtok1|Bearer' /tmp/server.log | head
```

Expected: `0` for the real token in every file; the wrong token never appears either (the gate logs
nothing per attempt). UI → Logs (`GET /0/logs`): any `[a-z2-9]{8}` word is `********`. Mode B: the
token is in the `Authorization` header, never the path, so `/tmp/exit0-wstunnel.log` shows
`/exit0/events` without a token. Pass: zero hits.

The exit laptop's own source was never touched, so §8 can proceed immediately. (`127.0.0.1` stays
locked for up to 15 min; nothing in this checklist needs it.)

---

## 8. Token regeneration

Client connected (Mode A), consumer browsing.

### 8.1 Regenerate

```sh
kvm# OLD=$TOKEN; grep -o '"token": *"[a-z2-9]*"' /etc/kvm/exit/0.json
```

Exit browser: Exit panel → Token → **Regenerate** → confirm (the UI tells you to stop the old client
first: do **not**, that is the test).

```sh
kvm# grep -oE '"token(CreatedAt)?": *"[^"]*"' /etc/kvm/exit/0.json; grep Authorization /etc/kvm/exit/0/wstunnel-restrict.yml | sed "s/$OLD/<OLD>/"
kvm# grep -E 'exit0.*(token regenerated|closed)' /tmp/server.log | tail -2
```

Expected: a new 8-char token and a new `tokenCreatedAt`; the restrict yaml carries the **new** token
(atomic rewrite; in Mode B `/tmp/exit0-wstunnel.log` shows the config reload, no restart:
`cat /var/run/exit0-wstunnel.pid` unchanged); server log: session closed with reason
`token regenerated`.

### 8.2 Old session terminated, old token refused

Exit window, within ~1 s: `session ended: websocket closed by kvm (1000 token regenerated)`,
`reconnecting in 1s`, then `connect failed: rejected (404): wrong token or slot, or this source is
rate limited`, `reconnecting in 2s` … `30s`, never `connected:` (Mode B: wstunnel client logs the
upgrade failing with 404, keeps retrying). UI: tunnel **disconnected** and stays so.

```sh
exit$ curl -sk -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $OLD" https://$KVM/exit/0/client.sh    # 404
```

Consumer: `dig example.com +noall +comments | grep status` → `SERVFAIL`; `curl -m 5 https://1.1.1.1/`
fast refusal (§4.3). Pass: old token dead everywhere, downstream still closed.

### 8.3 New token required, and the limiter was reset for the old peer

Ctrl+C the old client (it has been failing: its source accumulated failures). Copy the **new** command
from the panel and paste it. Expected: `connected:` **immediately** (RegenerateToken called
`limiter.Reset(peerSource)` for the connected peer, so the failed retries after the regen did not
lock it out — if this shows `404` for a while, note how long: that is the limiter, and it means the
reset happened before the retries; file it). UI connected, `previousPeer` unchanged (it records the
last change of peer, not a live replacement).
An old client left retrying with the dead token counts each attempt against its source, so the
operator's own machine can be locked out of the new token for minutes (the lockout doubles per
failure); stop the old client before or right after regenerating.
Consumer internet back (§5.1 check). Pass: new command connects, consumer resumes.

Check shell history hygiene note: the old one-liner is still in `history` on the exit; the panel says
so (D10).

---

## 9. Stale connection replacement

Needs two exit sources: exit A (the laptop, address `A`) and exit B (second laptop, or a phone hotspot
giving the same laptop a different address, address `B`). Client A connected, consumer browsing.

### 9.1 Make A unresponsive without closing its socket

Option 1 (cleanest, on the NanoKVM): silently drop A's packets.

```sh
kvm# A=<exit A address>; iptables -I INPUT 1 -s $A -p tcp -m multiport --dports 80,443 -j DROP; date +%T
```

Option 2: close the lid of exit A (sleep), or pull its Ethernet without stopping the client.

Expected on the NanoKVM: nothing changes for up to 100 s (`PingInterval × (PingMisses+2)` read
deadline). UI still says connected with peer A. Consumer: curl hangs/times out (not refused: the
front door still has a backend; streams get RST only when the WS is declared dead). Note the time.

### 9.2 B connects and supersedes A immediately

```sh
exit-B$ <paste the same command (same token)>
```

Expected within ~1 s: B prints `connected:`; UI: peer becomes B, **`previousPeer` = A with
`peerChangedAt`**, the panel shows the "changed from A" notice; server log:
`exit0: … superseding session from <A> with <B>` at warn level (Mode B: `wstunnel peer <A> superseded by <B>`).
A names its replacement as it goes: `session ended: websocket closed by kvm (1000 superseded by
<B addr:port>)`, then `reconnecting in 1s`. `client.ps1` prints the close code by name,
`NormalClosure superseded by <B addr:port>`, and mixes in its own `websocket state CloseReceived` and
`websocket send failed: An established connection was aborted by the software in your host machine`
when the close lands mid-send. Observed ping-pong rate with two live clients: a swap every 1 to 2 s.
Consumer, no action: `curl -4 -sS -o /dev/null -w '%{http_code}\n' https://example.com/` → `200`
through B (`curl https://api.ipify.org` shows **B's** public IP).

```sh
kvm# iptables -D INPUT -s $A -p tcp -m multiport --dports 80,443 -j DROP     # only if Option 1
```

### 9.3 The stale one wakes up

Wake A / restore its network. Expected on A: `session ended: … (i/o timeout or websocket closed)`
then `reconnecting in 1s` → **`connected:`** — and it supersedes B (D12: newest valid connection
wins; B logs `superseded by a new exit` and reconnects after 1 s, and the two then take turns every
second). This ping-pong is by design; stop one of them. Record it. Consumer stays online through
whichever holds the slot (a few hundred ms hiccup per swap).

### 9.4 Without a replacement: dead-peer detection

Reconnect A alone, then Option 1 again, and do **not** connect B:

```sh
kvm# date +%T; while :; do sleep 5; grep -q 'exit0.*ping\|deadline\|timeout' /tmp/server.log && { date +%T; grep -E 'exit0.*(ping|deadline|timeout|closed)' /tmp/server.log | tail -1; break; }; done
```

Expected: the kvm closes the session after 3 missed pings, between 60 s and 100 s after the drop; UI
→ disconnected; consumer → SERVFAIL/refusal (§4.3), no leak; every hev stream got RST (the hung curl
on the consumer ends with a reset at that moment). Remove the DROP rule; A reconnects with backoff
(A's own 75 s no-frame rule may fire first — either is a pass). Pass: dead peer detected ≤ 100 s,
supersede ≤ 1 s.

### 9.5 pinPeer

Advanced → pinPeer on (`POST /0/config {pinPeer:true}`), A connected. B pastes the command.
Expected: B gets `rejected (404)` (Mode A mux `PinPeer refuses a different RemoteAddr`; Mode B
`refusing wstunnel from B, pinned to A` in the server log), A stays connected, consumer unaffected.
pinPeer off → B's next retry supersedes. Turn pinPeer off.

---

## 10. MTU and MSS

### 10.1 What is configured

```sh
kvm# ip link show exit0 | grep -o 'mtu [0-9]*'; grep mtu /etc/kvm/exit/0/hev.yml; grep MTU= /etc/kvm/exit/0/env
kvm# iptables -S EXIT0 | grep TCPMSS
kvm# ip link show $NIC | grep -o 'mtu [0-9]*'
```

Expected: `mtu 1280` ×3; two TCPMSS rules with `--set-mss 1240`; the gadget NIC itself stays at 1500
(the clamp, not the NIC MTU, does the work).

### 10.2 MSS seen by the consumer

Start a capture on the consumer, then open one HTTPS connection.

```powershell
win> # Wireshark on the NanoKVM adapter, display filter:  tcp.flags.syn==1   → look at tcp.options.mss_val in both the SYN and the SYN/ACK
win> # or, without Wireshark:  pktmon filter add -t tcp; pktmon start -c --pkt-size 0 -f C:\pm.etl; curl.exe -s -o NUL https://www.iana.org/; pktmon stop; pktmon etl2txt C:\pm.etl -o C:\pm.txt; findstr /i "mss" C:\pm.txt
```

```sh
mac$ sudo tcpdump -ni $IF -c 4 -vv 'tcp[tcpflags] & tcp-syn != 0' 2>/dev/null | grep -oE '(Flags \[S[.]?\]|mss [0-9]+)' & sleep 1; curl -4 -sS -o /dev/null https://example.com/; wait
lin$ sudo tcpdump -ni $IF -c 4 -vv 'tcp[tcpflags] & tcp-syn != 0' 2>/dev/null | grep -oE '(Flags \[S[.]?\]|mss [0-9]+)' & sleep 1; curl -4 -sS -o /dev/null https://example.com/; wait
```

Expected: the consumer's SYN carries its own `mss 1460` (Windows/Linux) or `1440`/`1460` (macOS);
the SYN/ACK arriving at the consumer carries **`mss 1240`** (clamped by the `-i exit0 -o $NIC` rule;
without it lwip would advertise 8191). Then, mid-transfer, no data segment from either side exceeds
1240 bytes of payload: `tcpdump -ni $IF -c 50 'tcp and greater 1300'` during 10.3 prints nothing.

```sh
kvm# tcpdump -ni $NIC -c 2 -vv 'tcp[tcpflags] & (tcp-syn|tcp-ack) == (tcp-syn|tcp-ack)' | grep -o 'mss [0-9]*'   # if tcpdump exists: same 1240 on the way out
kvm# iptables -L EXIT0 -v -n -x | grep TCPMSS   # both counters advanced by one per connection
```

Pass: `mss 1240` on the SYN/ACK; both TCPMSS counters move.

### 10.3 Large download (≥ 200 MB)

```powershell
win> curl.exe -4 -sS -o NUL -w "%{http_code} %{size_download} bytes %{speed_download} B/s %{time_total}s`n" "https://speed.cloudflare.com/__down?bytes=200000000"
```

```sh
mac$ curl -4 -sS -o /dev/null -w '%{http_code} %{size_download} bytes %{speed_download} B/s %{time_total}s\n' 'https://speed.cloudflare.com/__down?bytes=200000000'
lin$ curl -4 -sS -o /dev/null -w '%{http_code} %{size_download} bytes %{speed_download} B/s %{time_total}s\n' 'https://speed.cloudflare.com/__down?bytes=200000000'
```

Also one integrity-checked file (any size ≥ 100 MB with a published SHA-256, e.g. a Linux ISO
netinst image and its `SHA256SUMS`): `curl -4 -sSLo /tmp/f.iso <url>; shasum -a 256 /tmp/f.iso`
(Windows: `Get-FileHash`).

Expected: `200 200000000 bytes`, no stall (watch `curl -#` progress if in doubt: a black hole
shows as headers arriving and the body freezing at a few KB, or at exactly a multiple of ~1240;
`--speed-limit 1000 --speed-time 15` makes curl abort on such a stall). Throughput is bounded by
`streamWindow / RTT` (128 KiB / RTT: ≈ 2.5 MB/s at 50 ms) in Mode A per stream, and by the exit's
uplink; record `speed_download`, RTT to the exit, and the exit's own `speed_download` for the same URL
for comparison. The checksum matches.

```sh
kvm# ip -s link show exit0 | sed -n '3,6p'; grep -E 'Ip:' -A1 /proc/net/snmp | awk 'NR==2{for(i=1;i<=NF;i++)h[i]=$i} NR==4{for(i=1;i<=NF;i++) if(h[i]~/Frag|Reasm|InHdrErrors|OutNoRoutes/) print h[i], $i}'
kvm# grep -ciE 'error|fail' /tmp/exit0-hev.log
```

Expected: `exit0` errors/dropped `0`; `FragFails 0`, `ReasmFails 0`; no new hev errors. RSS check of
hev during the transfer (3.9's command) stays flat.

`speed.cloudflare.com/__down` answers `403` to both the consumer and the exit, so it measures nothing
here; use a published image with a checksum instead. Measured on this unit, Windows consumer, Mode A,
`alpine-standard-3.21.0-x86_64.iso`: `http=200 size=251658240 speed=688097 time=365.7`, SHA-256 equal
to the published one, against `speed=2643939` for the same file fetched by the exit device itself, so
roughly a quarter of the exit's own throughput. Through the whole transfer hev RSS sat at 2360 kB and
fell to 2320 kB, `exit0` counted 0 errors and 0 dropped, every `Frag` and `Reasm` counter in
`/proc/net/snmp` stayed 0, and `/tmp/exit0-hev.log` stayed empty.

### 10.4 Upload

```powershell
win> fsutil file createnew $env:TEMP\big.bin 209715200; curl.exe -4 -sS -o NUL -w "%{http_code} %{size_upload} bytes %{speed_upload} B/s`n" -X POST --data-binary "@$env:TEMP\big.bin" https://speed.cloudflare.com/__up
```

```sh
mac$ head -c 200M /dev/urandom > /tmp/big.bin; curl -4 -sS -o /dev/null -w '%{http_code} %{size_upload} bytes %{speed_upload} B/s\n' -X POST --data-binary @/tmp/big.bin https://speed.cloudflare.com/__up
lin$ head -c 200M /dev/urandom > /tmp/big.bin; curl -4 -sS -o /dev/null -w '%{http_code} %{size_upload} bytes %{speed_upload} B/s\n' -X POST --data-binary @/tmp/big.bin https://speed.cloudflare.com/__up
```

Expected: `200 209715200 bytes` (`200 200000000` on macOS/Linux depending on `head -c 200M`), no
stall. The upload exercises the consumer → exit0 → SOCKS → WS direction and the kvm-side WINDOW
credit; if it stalls at exactly 128 KiB or 4 MiB, that is a flow-control bug, not MTU — file with the
client's output. `__up` does answer `200`, unlike `__down` above. Measured on this unit, Windows
consumer, Mode A, 20 MiB: `http=200 sent=20971520 speed=582186 time=36.0`, the exact byte count and
no stall at 128 KiB or 4 MiB. Pass: both directions complete with the exact byte count.

### 10.5 Repeat 10.3 in Mode B

Switch to wstunnel (§2), reconnect, rerun 10.3 and 10.4. Expected: same outcome; throughput is not
windowed by nexit so it may be higher. Switch back to native.

---

## 11. DNS end to end, no leak

Client connected. On the NanoKVM start the capture (or the fallback counters) first.

### 11.1 Nothing to port 53 leaves the NanoKVM

```sh
kvm# ip -br link | grep -E '^(eth0|br0|wlan0|tailscale0)'
kvm# tcpdump -nli eth0 -c 50 'port 53' 2>/dev/null | tee /tmp/dns-eth0.txt &   # add -i br0 / wlan0 too if they exist
```

Consumer, a unique name each time so caches cannot hide anything:

```powershell
win> nslookup leaktest-$(Get-Random).example.com
win> nslookup www.wikipedia.org
```

```sh
mac$ dig leaktest-$RANDOM.example.com +short; dig www.wikipedia.org +short
```

```sh
kvm# sleep 5; kill %1 2>/dev/null; wc -l < /tmp/dns-eth0.txt; grep -i leaktest /tmp/dns-eth0.txt
```

Expected: **zero** lines mentioning `leaktest`, and ideally zero port-53 packets at all (any that
appear are the NanoKVM's own — avahi, tailscale, the server's own lookups — and must not carry the
consumer's names). Without tcpdump: `netstat -anu | grep ':53'` shows no sockets to an external
resolver, and `(conntrack -L || cat /proc/net/nf_conntrack) | grep 'dport=53'` shows only
`dst=$GW` entries (the DNAT'd ones) and nothing with `dst=1.1.1.1`/`8.8.8.8` on the NanoKVM's own
side.

Where the query **should** appear is the exit device:

```sh
exit$ sudo tcpdump -nli any -c 10 'tcp port 53' &   # the forwarder queries 1.1.1.1 / 8.8.8.8 over TCP through the tunnel
mac$ dig leaktest-$RANDOM.example.com +short
```

Expected: TCP 53 to `1.1.1.1` (first upstream) from the exit device carrying the `leaktest` name.
Pass: name visible on the exit, invisible on eth0/br0.

### 11.2 The consumer's resolver is the gateway and nothing else

```powershell
win> Get-DnsClientServerAddress -AddressFamily IPv4 | ? ServerAddresses | ft InterfaceAlias,ServerAddresses
win> Get-DnsClientGlobalSetting | ft UseSuffixSearchList,SuffixSearchList
win> ipconfig /displaydns | Select-String -Pattern 'leaktest' -Context 0,4
```

```sh
mac$ scutil --dns | grep -E 'nameserver|if_index|resolver #'
lin$ resolvectl status | grep -E 'DNS Servers|Current DNS|Link'
```

Expected: only `10.a.b.1` for the NanoKVM interface (other interfaces are down). Windows Smart
Multi-Homed Name Resolution will race all adapters if Wi-Fi is up: keep it down for this step.

### 11.3 Hardcoded resolver is DNAT'd

```sh
kvm# iptables -t nat -L EXIT0_NAT -v -n -x | awk '$3=="DNAT"{print $1, $NF}'; $S status 0 | grep dns_redirected
```

```powershell
win> nslookup www.iana.org 8.8.8.8
win> Resolve-DnsName -Server 9.9.9.9 -Type A www.iana.org | ft Name,IPAddress
```

```sh
mac$ dig @8.8.8.8 example.com +noall +answer +comments +stats | grep -E 'status|SERVER|IN A'
mac$ dig @1.1.1.1 +tcp example.com +short
mac$ dig @8.8.8.8 AAAA example.com +noall +comments +answer | grep -E 'status|ANSWER:'
```

```sh
kvm# iptables -t nat -L EXIT0_NAT -v -n -x | awk '$3=="DNAT"{print $1, $NF}'; $S status 0 | grep dns_redirected
```

Expected: every query is answered (`status: NOERROR`, an A record) and the client believes it spoke
to `8.8.8.8#53` (`SERVER: 8.8.8.8#53(8.8.8.8)`: the DNAT is transparent, the reply is un-NATed); the
udp DNAT counter grows by the number of UDP queries, the tcp one by the `+tcp` query (if a watchdog
converge fell between the two readings the chain counters are lower, not higher: then trust
`dns_redirected`, which is `cat /var/run/exit0.dns_redirected` + the live chain counters);
`dns_redirected` grows by the same total (UI `dns.redirected`). AAAA via 8.8.8.8 is still an empty
NOERROR (it is our forwarder answering, not Google). `dig @$GW` queries do **not** count as redirected
(`! -d $GW`).

Disconnected control: Ctrl+C the exit client, `dig @8.8.8.8 example.com` → `SERVFAIL` immediately (the
forwarder, not a timeout to Google). Reconnect.

Not covered by the DNAT and expected to pass through untouched: DoH (443) and DoT (853) to public
resolvers — `curl -4 -sS 'https://1.1.1.1/dns-query?name=example.com&type=A' -H 'accept: application/dns-json'` works
and does **not** move the DNAT counters. Note it; it is by design.

### 11.4 Forwarder cache and counters

```sh
mac$ for i in 1 2 3; do dig example.com +noall +stats | grep 'Query time'; done
```

Expected: first query tens/hundreds of ms (through the tunnel), second and third `0 msec`-ish
(1000-entry positive cache, TTL capped at 300 s). UI `dns.queries` +3, `dns.failures` 0. Pass: all
of 11.1–11.4.

---

## 12. Coexistence with keyboard, mouse, video, mass storage, UVC

### 12.1 Baseline with the slot enabled

Profile: `ls /boot/usb.*` from 0.1. Consumer, list the composite:

```powershell
win> Get-PnpDevice -PresentOnly | ? InstanceId -like 'USB\VID_*' | ? FriendlyName -match 'HID|Keyboard|Mouse|NCM|RNDIS|Ethernet|Mass Storage|Disk|Camera|Video' | ft Class,FriendlyName,Status
win> Get-Disk | ? BusType -eq USB | ft Number,FriendlyName,Size,OperationalStatus
```

```sh
mac$ system_profiler SPUSBDataType 2>/dev/null | grep -A12 -i 'nanokvm\|sipeed\|gadget' | grep -E 'Product ID|Vendor ID|Serial|Speed|BSD Name'; ioreg -p IOUSB -l -w0 | grep -E '"USB Interface Name"|bInterfaceClass' | sort | uniq -c
lin$ lsusb -d $(lsusb | grep -i -E 'sipeed|nanokvm|gadget|rndis|ncm' | awk '{print $6}' | head -1) -v 2>/dev/null | grep -E 'bInterfaceClass|iInterface'; lsblk -S | grep -i usb
```

Expected interface set = the profile: N × HID (keyboard, mouse, maybe consumer control), CDC NCM
(or RNDIS as two interfaces), Mass Storage if `usb.disk0`, Video Control + Video Streaming if the
UVC sentinel is present. Then, from the exit browser's KVM view with the consumer's desktop visible:

- type `echo exit-check-1` into a terminal on the consumer: appears exactly;
- move the mouse in a circle, click, right-click, scroll: all track;
- video: the stream stays live at the panel's fps, no black frames;
- mass storage (if in profile): the virtual disk is mounted/listed on the consumer (`Get-Disk`,
  Finder, `lsblk`); read a file from it;
- UVC (if in profile): open the camera in Windows Camera / Photo Booth / `ffplay /dev/video*`: frames.

All while §3.3's curl loop runs on the consumer (`while :; do curl -4 -sS -o /dev/null https://example.com/; sleep 1; done`).
Pass: nothing degrades with traffic flowing.

### 12.2 The disable/enable rebind cycle

Exit browser: Exit switch **off** → confirm.

Expected: consumer HID drops for ~2 s (a held key does not stick), the NIC disappears and comes back
with a lease that has **no** Default Gateway and **no** DNS Server:

```powershell
win> ipconfig /all | Select-String -Context 0,12 'Gadget|NCM|RNDIS'
```

```sh
mac$ ipconfig getpacket $IF | grep -E 'router|domain_name_server|yiaddr'
lin$ nmcli -f DHCP4 dev show $IF | grep -E 'routers|domain_name_servers' || echo no-router-no-dns
```

```sh
kvm# grep -E '"(enabled|pending)"' /etc/kvm/exit/0.json; ls /etc/kvm/exit/gadget.route 2>&1
kvm# $S status 0; echo rc=$?
kvm# ip rule | grep -E '^100[01]:'; ip route show table 100; ip link show exit0 2>&1
kvm# iptables -S | grep -c EXIT0; iptables -t nat -S | grep -c EXIT0; ip6tables -S FORWARD | grep -c "$NIC"
kvm# netstat -lntp | grep -E ':10800|:10810|:10820|:53 '; ps w | grep '[h]ev'; ps w | grep '[w]stunnel'; ps w | grep -c '[u]dhcpd'
kvm# tail -2 /etc/udhcpd.$NIC.conf
kvm# iptables -S FORWARD; iptables -t nat -S PREROUTING; iptables -t nat -S POSTROUTING     # compare with 0.4's baseline
```

Expected: `"enabled": false`, `"pending": false`, no marker; `status` prints `forward=…` with
`routing=0 tun=0 hev=0 nat=0`, `rc=1`; no `100x` rules, empty table 100, `exit0` `does not exist`;
`0 0 0` chain references; no listeners, no hev, exactly one udhcpd; the conf ends with `option lease
864000` (no `opt router`/`opt dns`); the FORWARD/PREROUTING/POSTROUTING listings equal 0.4's
baseline byte for byte. `ip_forward` may stay `1` (S94exit does not lower it; other services own it).
Consumer, with Wi-Fi back on: internet via Wi-Fi as before; the NanoKVM UI at `10.a.b.1` still loads.
The exit client (if still running) logs `rejected (404)` and retries.

Then switch **on** again. Expected: §1.1 state exactly, one udhcpd, consumer lease with
gateway/DNS again, HID/mouse/video/disk/camera all back (repeat the 12.1 actions), the exit client
reconnects on its own and §3.3 passes.

```sh
kvm# cat /sys/class/udc/*/state; grep -c . /var/lib/misc/udhcpd.$NIC.leases 2>/dev/null; dmesg | tail -20 | grep -iE 'dwc2|gadget|configfs|usb' | tail -5
```

Expected: `configured`; no gadget errors in dmesg.

### 12.3 Three cycles

Repeat off/on three times, 30 s apart, running only the quick checks after each:
`$S status 0 | tr '\n' ' '; ps w | grep -c '[u]dhcpd'; cat /sys/class/udc/*/state; ipconfig-equivalent gateway present?; keyboard types; curl 200`.

Expected: identical each time, one udhcpd, UDC `configured`, no leaked `EXIT0` rules or stale `exit0`
in the off state, no orphaned `hev` (`pgrep hev` empty when off). If the panel ever shows
`re-plug the USB cable to pick up the new gateway: …`, record the reason text (functionfs session,
UDC loaned to passthrough, media busy), replug once, and confirm the lease refreshes.

Windows consumer, after each cycle:

```powershell
win> Get-PnpDevice -PresentOnly | ? InstanceId -like 'USB\VID_1D6B*' | ft Status,Class,FriendlyName
```

Expected: every interface `OK`, the two audio nodes included. An audio node at `Error` with problem
code 10 means the host's first UVC class request beat the video node's event subscription and ep0 is
wedged; nothing on the host recovers it, only another gadget rebind does.

### 12.4 Bridge exclusion

With the slot enabled, try to enable the L2 bridge in its own panel. Expected: refused with a message
naming exit slot 0. Disable exit, enable the bridge, try to enable exit: refused with `the gadget NIC
belongs to the L2 bridge: br0 exists` (or the LKG/uplink reason). Disable the bridge; end in the
stock state (exit off, bridge off) unless the unit is meant to stay in exit mode.

---

## What to capture when something differs

Run all of this on the NanoKVM the moment a step fails, **before** disabling, re-enabling, rebooting
or re-running the client; attach the bundle and the failing step's consumer/exit output verbatim.

```sh
kvm# D=/tmp/exit-capture-$(date +%s); mkdir -p $D; cd $D
kvm# cp /etc/kvm/exit/0.json 0.json && sed -i 's/"token": *"[^"]*"/"token":"<redacted>"/' 0.json; cp /etc/kvm/exit/0/env /etc/kvm/exit/0/hev.yml /etc/kvm/exit/0/nic /etc/kvm/exit/0.state.json . 2>/dev/null; sed "s/Bearer [a-z2-9]*/Bearer <redacted>/" /etc/kvm/exit/0/wstunnel-restrict.yml > wstunnel-restrict.yml 2>/dev/null; cat /etc/kvm/exit/gadget.route > gadget.route 2>&1
kvm# /etc/init.d/S94exit status 0 > s94-status.txt 2>&1; echo "rc=$?" >> s94-status.txt
kvm# ip rule > ip-rule.txt; ip route show table 100 > table100.txt 2>&1; ip route > ip-route-main.txt; ip -d link > ip-link.txt; ip -4 addr > ip-addr.txt; ip -s -s link > ip-s-link.txt
kvm# iptables -S > iptables-S.txt; iptables -S EXIT0 > EXIT0.txt 2>&1; iptables -t nat -S EXIT0_NAT > EXIT0_NAT.txt 2>&1; iptables -t nat -S EXIT0_MASQ > EXIT0_MASQ.txt 2>&1; iptables -L EXIT0 -v -n -x > EXIT0-counters.txt 2>&1; iptables -t nat -L -v -n -x > nat-counters.txt; ip6tables -S > ip6tables-S.txt 2>&1
kvm# sysctl net.ipv4.ip_forward net.ipv4.conf.all.rp_filter net.ipv4.conf.exit0.rp_filter net.ipv4.conf.$NIC.route_localnet net.ipv6.conf.$NIC.disable_ipv6 > sysctl.txt 2>&1
kvm# cp /tmp/exit0-hev.log /tmp/exit0-wstunnel.log . 2>/dev/null; tail -500 /tmp/server.log > server-log-tail.txt 2>/dev/null; grep -E 'exit0|exit: slot' /tmp/server.log > server-log-exit.txt 2>/dev/null
kvm# (conntrack -L 2>/dev/null || cat /proc/net/nf_conntrack) > conntrack.txt 2>&1
kvm# netstat -lntup > netstat-listen.txt 2>&1; netstat -ntu > netstat-conns.txt 2>&1
kvm# ls -l /var/run/exit0* /etc/kvm/bin/ > runfiles.txt 2>&1; cat /var/run/exit0.dns_redirected > dns_redirected.txt 2>&1; for p in $(cat /var/run/exit0-hev.pid /var/run/exit0-wstunnel.pid 2>/dev/null) $(pidof NanoKVM-Server); do echo "== $p $(tr '\0' ' ' < /proc/$p/cmdline)"; grep -E 'VmRSS|VmSize|VmPeak|Threads' /proc/$p/status; done > procs.txt 2>&1
kvm# ps w | grep '[u]dhcpd' > udhcpd.txt; cp /etc/udhcpd.$NIC.conf . 2>/dev/null; xxd /var/lib/misc/udhcpd.$NIC.leases > leases.hex 2>/dev/null
kvm# cat /sys/kernel/config/usb_gadget/g0/UDC > udc.txt; cat /sys/class/udc/*/state >> udc.txt; ls /sys/kernel/config/usb_gadget/g0/configs/c.1/ >> udc.txt; cat /sys/kernel/config/usb_gadget/g0/configs/c.1/*/ifname >> udc.txt 2>&1
kvm# dmesg | tail -200 > dmesg-tail.txt; free -m > free.txt; uptime > uptime.txt; zcat /proc/config.gz | grep -E 'CONFIG_(TUN|IP_MULTIPLE_TABLES|NFT_COMPAT|NETFILTER_XT_TARGET_TCPMSS|IP_NF_TARGET_REJECT)=' > kconfig.txt
kvm# tcpdump -ni $NIC -c 200 -w gadget.pcap 2>/dev/null & tcpdump -ni eth0 -c 200 -w eth0.pcap 'not port 22' 2>/dev/null &      # then reproduce the failing consumer command, wait 10 s, `kill %1 %2`
kvm# cd /tmp && tar czf $D.tgz $(basename $D) && ls -l $D.tgz     # scp it off
```

From the **consumer**, alongside: the full output of the failing command, `ipconfig /all` /
`ipconfig getpacket $IF` / `nmcli dev show $IF`, `Get-NetRoute` / `netstat -rn` / `ip route`,
`nslookup`/`dig` with `+comments +stats`, and for MTU/MSS issues the pcap from 10.2 (Wireshark
`File → Export Specified Packets`, or the tcpdump `-w` file).

From the **exit device**: the client's entire terminal output from paste to failure (it prints
`session ended … reason` and every `reconnecting in`), `curl -sk -i` of the failing URL with the
`Date:` line removed, and for Mode B the wstunnel client log (`RUST_LOG=info` if terse).

From the **browser** (exit device, the UI): DevTools → Network → filter `exit` → save
`/api/extensions/exit/0/status` and `/commands` responses as JSON (right-click → Copy response), and
the Console for any `settings.exit.*` raw i18n keys or React errors; a screenshot of the Exit panel's
status card with the timestamp visible.

State in the report, for each step: pass/fail, the exact expected-vs-observed text, Mode A or B,
consumer OS and interface driver (NCM vs RNDIS), exit OS and runtime (python3/perl/PowerShell 5.1),
image (`FULL`/`MINIMAL`, kernel from 0.1), and the uptimes from the 0.7/6.1 timelines.

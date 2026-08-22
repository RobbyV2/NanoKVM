# Ethernet bridge

Status: design only. No code written, nothing has run on hardware.
Date: 2026-08-22.

## Goal

Put `eth0` and the USB gadget NIC into one Layer-2 bridge, `br0`, so the router on the other end of the Ethernet cable sees the NanoKVM and the controlled host as two separate devices, each with its own MAC and its own DHCP lease from the network's existing server. No NAT, no address translation, no second DHCP server, no routing between subnets. Frames arriving on `eth0` addressed to the host's MAC are forwarded out `usb0` unmodified, and frames the host sends go out `eth0` with the host's own source MAC.

Program 3 depends on Program 1. `/etc/init.d/S03usbdev stop` is `echo host > /proc/cviusb/otg_role` (`S03usbdev:162-163`), so it drops the entire gadget including the HID keyboard and mouse. Enslaving and releasing `usb0` without losing the console requires something that can add and remove a single function symlink around a bind and unbind, and that is the presentation manager.

Program 3 also requires a custom base image, which neither of the other two programs does. The stock NanoKVM kernel has no bridge support at all: a repo-wide grep for `brctl`, `bridge-utils`, `CONFIG_BRIDGE`, `br_netfilter`, `ip link set ... master`, `macvlan` and `vlan` returns zero hits, there is no `/etc/network/interfaces` anywhere, and `S00kmod` insmods only `soph_*.ko` SoC drivers from `/mnt/system/ko/`.

## Why the obvious approach fails

The obvious shape is one init script between `S03usbdev` and `S30eth` that creates `br0`, enslaves `eth0` and `usb0`, and lets everything downstream carry on unchanged. Each of the five things that breaks breaks silently.

`S30eth:26` is `ip addr flush dev eth0`, unconditional, at every `start`. A bridge created behind the script's back gets its address stripped by a script that then re-adds it to `eth0` at `:53` and puts the default route on `eth0` at `:54`, so the address lives on a port and the route points at a device with no address. Nothing logs any of this.

The DHCP branch at `S30eth:68` is `udhcpc -i eth0 -t 10 -T 1 -A 5 -b`, backgrounded, with no fallback of any kind. `RESERVE_INET="192.168.90.1/22"` exists only inside the `/boot/eth.nodhcp` branch at `:59-66`. A bridge holds carrier down until a port forwards, and with STP that is a listening plus learning delay of roughly thirty seconds, so `udhcpc` exhausts ten one-second tries and exits, leaving the device with no IPv4 and nothing that will retry.

Every consumer of interface state matches by name against a hardcoded string or prefix. `ip_address()["eth0"]` appears three times in `system_state.cpp`, `ping -I eth0` once, an `ip route | grep -i 'eth0'` once, and both Go classifiers accept only `eth`, `en`, `wlan` and `wl` prefixes and discard everything else. A device named `br0` is invisible to all of them, and the OLED pins itself at `eth_state = 1` forever.

All five firewall rules at `S95nanokvm:92-105` are hardcoded `-i eth0` or `-o eth0`, including `DROP OUTPUT tcp --sport 8000`, which keeps the raw stream port off the wire. Bridged frames that the bridge forwards from port to port never reach iptables at all unless `br_netfilter` is loaded.

And `S03usbdev stop` tears out the whole gadget, so the existing runtime toggle at `virtual-device.go:21-32` would drop `usb0` out from under the bridge and take keyboard and mouse with it.

## Decisions

| # | Decision |
|---|---|
| D1 | The kernel now supports it. `CONFIG_BRIDGE=y` was added to the LicheeRV-Nano-Build fork at commit `f74b732` on branch `nanokvm-custom`, branched from upstream's `NanoKVM` at `1559c57`. Built in rather than a module, because nothing in the boot path establishes that a module can be loaded at all. It auto-selects `LLC` and `STP`, and neither is written into the defconfig: both are promptless tristates, a `select` from a built-in symbol promotes them to `y`, and `savedefconfig` drops the redundant lines again. `BRIDGE_NETFILTER` is deliberately not enabled. Userspace needs nothing new, since `BR2_PACKAGE_IPROUTE2=y` already ships both `ip` and `bridge`. Program 3 therefore requires a custom base image. Detail below. |
| D2 | Ownership handoff, not coexistence. `S30eth` gains one indirection, reading the uplink interface name from `/etc/kvm/network/l2-uplink` with `eth0` as the fallback, used for its flush, its `arping`, both `dev` targets and its `udhcpc -i`. `br0` is never created behind the script's back. |
| D3 | `br0` is created with `eth0` enslaved unconditionally, before `S30eth` runs. `usb0` is enslaved and released dynamically by the presentation manager as part of a profile apply, never by an init script. This is the dependency on Program 1. |
| D4 | STP off, not merely fast. `ip link set br0 type bridge stp_state 0` for a two-port bridge with no loop risk. Getting this wrong makes the device unreachable, and the DHCP branch has no `RESERVE_INET` fallback. |
| D5 | Pin the bridge MAC to `eth0`'s permanent address at creation, so enslaving or releasing `usb0` never flips the bridge's L2 identity and never breaks a DHCP reservation. |
| D6 | Fix the latent `grep inet` bug at `S30eth:59,61` while in there. Use `ip -4`. Enslaving changes address timing enough to make the latent bug live. |
| D7 | Two escape hatches are used rather than fought: write the current default gateway to `/etc/kvm/gateway`, which `system_state.cpp` already prefers over its `ip route` grep, and patch the two Go interface classifiers to treat `br*` as wired plus the four `system_state.cpp` sites to read the uplink name from `/etc/kvm/network/l2-uplink`. Detail below. |
| D8 | The five `S95nanokvm` firewall rules are rewritten against the uplink name. `br_netfilter` is not loaded. Reasoning below. |
| D9 | Dead-man rollback. Snapshot to disk before the first mutation, arm a pending marker with a deadline before mutating, restore unless disarmed, with a boot-time check so a power cut inside the window also recovers. Verification before disarming requires all three of an IPv4 address on `br0`, a default route through it whose gateway answers, and an inbound liveness proof. `wlan0` is never enslaved. |
| D10 | The gadget network protocol is a USB profile choice, not a bridge one, and it is NCM or RNDIS. ECM is not offered: there is no `f_ecm` branch in `S03usbdev`, no `ecm` `FunctionKind`, no compile case and no capability entry anywhere in the tree, so a three-way selector would offer a mode the gadget layer cannot build. The two that exist are surfaced on the existing Virtual Network control under Settings, Device, driven by the presentation profile, because the choice decides what the gadget presents to the attached host whether or not a bridge exists. The bridge names the active protocol in its panel, read from the presentation snapshot, and offers no control that would duplicate that one. It enslaves whichever network function is present. |

## Kernel and image

`CONFIG_BRIDGE=y` puts the bridge in the kernel image itself, and `CONFIG_BRIDGE` selects `CONFIG_LLC` and `CONFIG_STP`, so both are built in alongside it. Neither appears in the defconfig, since a `select` from a built-in symbol already promotes a promptless tristate to `y` and `savedefconfig` drops the line again. Nothing else in the buildroot config changes. `bridge-utils` is not added, because `BR2_PACKAGE_IPROUTE2=y` already puts `/sbin/ip` and `/sbin/bridge` on the image, and `ip link add name br0 type bridge` plus `ip link set eth0 master br0` covers everything `brctl` would have done.

`BRIDGE_NETFILTER` stays off for two reasons. A transparent Layer-2 bridge does not want netfilter hooks in the bridge forwarding path, since the entire point of the program is that the controlled host's frames reach the router unmodified and unfiltered. And the device has roughly 90 MB of usable RAM, so a conntrack pass over every forwarded frame is a cost paid on the wrong traffic.

Built in rather than a module because nothing on this image can be relied on to load one. `S00kmod` insmods a fixed list of 21 vendor `soph_*.ko` files by absolute path from `/mnt/system/ko/`, with no `modprobe`, no `depmod` and no wildcard, so neither an explicit `modprobe bridge` nor rtnetlink's own `request_module("rtnl-link-bridge")` autoload has ever run here, and `/lib/modules/$(uname -r)/modules.dep` is not known to be populated. Building the bridge in removes the load step, the init-time code that would perform it, and a failure mode on a device where no non-vendor module has ever been loaded. The cost is roughly 150 KB of kernel memory resident on devices that never bridge.

## What the network stack does today

Boot order is `S00kmod`, `S01fs` (mounts `/boot` vfat and configfs), `S03usbdev` (creates `usb0`, if and only if a network gadget flag is set), `S03usbhid`, `S15kvmhwd`, `S30eth`, `S30wifi`, the `S50` daemons, `S80dnsmasq` (a no-op on a stock image), `S95nanokvm` (firewall rules, then `kvm_system` and `NanoKVM-Server`), then the tunnel and Tailscale scripts. So `eth0` is addressed after `usb0` exists and before the firewall rules and either userspace daemon.

`usb0` has no IP configuration anywhere in the repo. `S03usbdev` writes `dev_addr` and `host_addr` and nothing else, never brings the interface up, and no script assigns it an address or starts a DHCP server on it. The only live DHCP server in the image is `udhcpd` on `wlan0` in AP mode (`S30wifi:103-121`), on a per-device subnet `10.<id3>.<id2>.1/24` derived from `sha512sum /sys/class/cvi-base/base_uid`. There is nothing on `eth0` or `usb0` for a bridge to collide with.

### `S30eth`, line by line

| Line | Today | Change |
|---|---|---|
| new, after `RESERVE_INET` | | `IFACE=$(cat /etc/kvm/network/l2-uplink 2>/dev/null \|\| echo eth0)` |
| `:26` | `ip addr flush dev eth0`, unconditional at every `start` | `dev "$IFACE"` |
| `:52` | `arping -Dqc2 -Ieth0 "$addr" \|\| continue`, the duplicate-address skip that makes `/boot/eth.nodhcp` a candidate list rather than a config line | `-I"$IFACE"` |
| `:53` | `ip a add "$inet" brd + dev eth0` | `dev "$IFACE"` |
| `:54` | `ip r add default via "$gw" dev eth0` | `dev "$IFACE"` |
| `:55` | `echo -e "nameserver $gw" >> /etc/resolv.conf`, an append, so duplicates accumulate on every restart | unchanged, and the reason the enable transaction replays a static address itself rather than re-running `start` |
| `:59,:61` | `ip a show dev eth0 \| grep inet`, which also matches `inet6` | `ip -4 a show dev "$IFACE" \| grep -q inet` |
| `:60` | `udhcpc -i eth0 -t 3 -T 1 -A 5 -b -p /run/udhcpc.eth0.pid` | `-i "$IFACE"`, pidfile path unchanged |
| `:65` | `ip a add "$inet" brd + dev eth0` with `RESERVE_INET="192.168.90.1/22"`, no route and no nameserver | `dev "$IFACE"` |
| `:68` | `udhcpc -i eth0 -t 10 -T 1 -A 5 -b -p /run/udhcpc.eth0.pid &`, backgrounded, no fallback at all | `-i "$IFACE"`, timers unchanged, because STP is off and carrier is immediate |
| `:73-75` | `stop` hard-exits 1 when `/run/udhcpc.eth0.pid` is absent, so on a static box `stop` never flushes | unchanged |

The pidfile path stays the literal `/run/udhcpc.eth0.pid`. It names the script's one DHCP client rather than an interface, and deriving it from `$IFACE` would make `stop` unable to find a client that `start` launched under the other name, and a live enable passes through exactly that state. `restart` still works on a static box, because `$0 stop` and `$0 start` are separate invocations and `stop`'s `exit 1` does not stop `start` from running.

`S30eth` never brings `eth0` up. It relies on the link already being up, or on `udhcpc` and `ip a add` doing it implicitly. `S29bridge` brings both `eth0` and `br0` up explicitly, so on a device whose PHY has not autonegotiated by S29 the link comes up earlier than it does today.

### The firewall

`S95nanokvm:92-105` installs five rules, each idempotent through `iptables -C ... || iptables -A ...`: ACCEPT INPUT tcp dport 80, ACCEPT OUTPUT tcp sport 80, ACCEPT INPUT tcp sport 22, ACCEPT OUTPUT tcp dport 22, and DROP OUTPUT tcp sport 8000. All ten interface matches are the literal `eth0`. An older duplicate of the same block lives at `kvmapp/jpg_stream/S95nanokvm:30-34`. Nothing in the tree runs `iptables-save` or `iptables-restore`, so the ruleset is rebuilt from scratch on every boot and there is no persisted state to migrate.

D8 rewrites the ten matches to `$IFACE`, read the same way `S30eth` reads it, and does not load `br_netfilter`. The reason this is sufficient is that `br0` is the L3 device once the bridge exists: a packet addressed to the NanoKVM's own IPv4 address arrives on `eth0`, is bridged, and is delivered locally through `br0`, so INPUT sees `-i br0`; a packet the server sends is routed out `br0`, so OUTPUT sees `-o br0`. The DROP rule keeps covering the raw stream port, and it now covers it on both physical paths at once, since a reply leaving toward the attached host is also routed out `br0`. `br_netfilter` would add hooks only for frames the bridge forwards from `eth0` to `usb0` and back, and those frames are the controlled host's own traffic to its own DHCP lease, which the program exists to leave alone.

`S95nanokvm` runs at S95, well after `S30eth` at S30, so `/etc/kvm/network/l2-uplink` is already written by the time the rules are installed. A live enable or disable re-installs the five rules against the new uplink and deletes the five copies naming the old one, because the boot-time idempotence check does not remove stale rules.

### AP mode

`S30wifi:116` is `ip route del default || true`, which deletes whatever default route exists, in practice the wired one. With `br0` carrying the default route, AP mode deletes that instead. This is unchanged behaviour and is recorded rather than fixed. `wlan0` is never enslaved into `br0`, since it is the out-of-band recovery path and `hostapd` plus bridging is its own category of problem.

## Interop with the `kvm_system` C++ observers and the Go classifiers

Every one of these matches an interface by name. The table is the full set.

| Site | Code | What breaks under a bridge |
|---|---|---|
| `system_state.cpp:55-56` | `strcmp(ip_address()["eth0"], kvm_sys_state.eth_addr)` in the `ETH_IP` case of `get_ip_addr` | the address is on `br0`, the map lookup returns empty, the function prints `can't get ip addr`, zeroes `eth_addr[0]` and returns 0 |
| `system_state.cpp:61-65` | copies 16 bytes of `ip_address()["eth0"]` into `kvm_sys_state.eth_addr` | the string the OLED renders is empty |
| `system_state.cpp:116` | `ip route \| grep -i '^default' \| grep -i 'eth0' \| awk '{print $3}'` under `popen`, reached only when `/etc/kvm/gateway` is absent | the default route is via `br0`, the grep returns nothing, `eth_route[0] == 0`, and `get_ip_addr(ETH_ROUTE)` returns 0 |
| `system_state.cpp:189` | `ping -I eth0 -w 1 <eth_route>` in `chack_net_state(ETH_ROUTE)` | binding the source to an enslaved port bypasses the bridge path, the gateway never answers, and `eth_state` cannot rise past 2 |
| `system_state.cpp:398` | `get_nic_state("eth0")` at the top of `kvm_update_eth_state()`, requiring both `IFF_UP` and `IFF_RUNNING` | survives as written, since an enslaved port keeps both flags, and is repointed anyway so that the state machine observes the device that carries the address |
| `system_state.cpp:402` | `strcmp(ip_address()["eth0"], kvm_sys_state.eth_addr)` change detection before `get_ip_addr(ETH_IP)` | empty on both sides forever, so no redraw is ever triggered |
| `system_state.cpp:402-407` | `kvm_update_eth_state()` sets `eth_state = 1` and early-returns when the NIC is RUNNING with no IPv4 | `eth_state` is pinned at 1 and can never reach 3 |
| `oled_ui.cpp:143` | `show_which_ip()` selects the wired IP only when `eth_state == 3 && wifi_state != 1` | the OLED alternates between wired and wireless forever, or shows the no-IP screen |
| `oled_ui.cpp:203-244` | `kvm_eth_state_disp()` latches `kvm_oled_state.eth_state` only when `eth_state >= 2 && eth_addr[0] != 0` | nothing is ever latched, so the wired panel never draws |
| `server/service/vm/ip.go:68-78` | `getInterfaceType` accepts `HasPrefix("eth"\|"en"\|"wlan"\|"wl")` and returns `Other` for everything else, which `GetInterfaceInfos()` drops | `br0` vanishes from `GET /api/vm/info`'s `ips[]` and the About panel at `about/information.tsx:51-57` renders blank |
| `server/service/network/dns.go:460-470` | `getDNSInterfaceType`, the same prefix set, returning `""` | `getDefaultIPv4Route()` reads the `00000000` row of `/proc/net/route`, gets `br0`, and the DNS panel shows an empty interface type |
| `system_state.cpp:485` | `get_nic_state("usb0")` in `kvm_update_rndis_state()` | inert, because `main.cpp:154` has the call commented out and `rndis_state` sits at its `-1` initializer, so nothing observes `usb0` at runtime |

Two escape hatches already exist in the C++ and are used rather than fought.

`/etc/kvm/gateway` is read at `system_state.cpp:111-151`, and when the file exists its contents are used verbatim instead of the `ip route` grep at `:116`. The bridge manager writes the current default gateway there on every apply and on every DHCP lease, through the same `99-nanokvm-dns` hook mechanism that `dns.go` already installs into `/usr/share/udhcpc/default.script.d/`. This fixes the gateway half on a device whose `kvm_system` binary has not been updated, which matters because an OTA can ship a new server without a new `kvm_system`.

The interface name cannot be fixed from a file, so `system_state.cpp` gains one helper that reads `/etc/kvm/network/l2-uplink` once with `"eth0"` as the fallback, used at the address lookup, the route grep, the ping and `get_nic_state`. The two Go classifiers each gain one line mapping a `br` prefix to wired. The alternative is renaming `eth0` to `eth0-phy` and calling the bridge `eth0`, which needs no consumer changes at all, and is rejected because it is far riskier at boot, with udev and naming races and `S30eth:26` flushing whichever device holds the name during the rename window, and because it makes the two devices indistinguishable in every log and every `ip` listing.

## Architecture

`/etc/kvm/network/l2-uplink` is a single line naming the interface that carries the management address. `dns.go:24-26` already owns `/etc/kvm/network/` for `dns.mode` and `dns.servers`, so the directory exists, is created by the server, and needs no new init-time `mkdir`. The file is the only shared state between the server, the two init scripts and the C++ daemon. Absent means `eth0`, the stock state, so an image that has never had the bridge enabled and an image that has had it disabled are byte-identical in this respect.

Two things create bridge state, and they never overlap. `S29bridge` runs at boot, between `S03usbdev` and `S30eth`, and when the last-known-good state says the bridge is enabled it creates `br0` with STP off, pins the MAC, enslaves `eth0` and brings both up. `S30eth` then flushes and addresses `br0` from the start, so there is no live handoff at boot and no window in which an address moves between devices. `S29bridge` also checks for an armed `pending.json` and, finding one, removes `l2-uplink` and leaves the bridge uncreated, so a power cut inside the apply window comes back on stock `eth0`.

The transactions in the server are the only thing that moves an address between devices, and they run only in response to an explicit API call.

Bridge state lives at `/etc/kvm/presentation/network/`, alongside the presentation manager's store, since the `usb0` half of the design is a presentation profile apply:

```
/etc/kvm/presentation/network/
  snapshot.json         # ip -j link/addr/route, resolv.conf, gateway, l2-uplink   0600
  pending.json          # snapshot path plus an absolute deadline                  0600
  last-known-good.json  # the last verified state, read by S29bridge at boot       0600
```

The directory is 0755 and every file is 0600, written through the `atomicFile` helper at `server/service/extensions/tunnel/config.go:78-152` verbatim, with a package-level `var networkDir` declared `var` rather than `const` so tests can swap it the way `tunnel/config_test.go:13-19` does. The Go lives in `server/service/presentation/network/`, importing the presentation manager for the one step that touches the gadget.

`br0` is created as `ip link add name br0 type bridge stp_state 0`. With `stp_state 0` the kernel's `br_make_forwarding()` puts a port straight into forwarding rather than through listening and learning, so `br0` raises carrier and reports `IFF_RUNNING` as soon as a port has carrier, instead of thirty seconds later. `forward_delay 0` is set as well and is redundant while STP is off, recorded here so that nobody later reads the redundancy as the load-bearing setting.

The MAC is pinned by reading `/sys/class/net/eth0/address` before enslaving anything and writing `ip link set br0 address <that>`. An explicit `br_dev_set_mac_address` marks the bridge address static, so `br_stp_recalculate_bridge_id()` stops re-electing the numerically lowest port address on every port change. Without the pin, enslaving `usb0` at `48:da:35:6e:xx:xx` can win that election depending on what `eth0`'s address is, the bridge's L2 identity changes under a live DHCP lease, and the stable-adapter guarantee that the `S03usbdev:42-45` comment exists to provide is broken from the other side.

## The enable transaction

Steps 1 through 12 hold the management address. Step 13 runs after the dead-man is disarmed, because releasing and enslaving `usb0` is reversible and never touches the management address.

1. Snapshot. `ip -j link show`, `ip -j addr show`, `ip -j route show`, the bytes of `/etc/resolv.conf`, the bytes of `/etc/kvm/gateway`, and the current `l2-uplink`. Write `snapshot.json` atomically and `fsync` it before the first mutation, so a power cut mid-apply leaves a record that boot-time code can act on.
2. Arm the dead-man. Write `pending.json` carrying the snapshot path and an absolute deadline sixty seconds out. A `context.WithTimeout` goroutine restores the snapshot unless disarmed, and `S29bridge` restores unconditionally at boot if it finds the file.
3. Create `br0` with `stp_state 0` and `forward_delay 0`, set its address to `eth0`'s permanent MAC read from `/sys/class/net/eth0/address`, and bring it up.
4. Kill any running `udhcpc` named by `/run/udhcpc.eth0.pid` and remove the pidfile, so no lease renewal re-adds an address to a device that is about to become a port.
5. Capture `eth0`'s current IPv4 addresses and its default route from the snapshot already taken, then `ip addr flush dev eth0` and delete its default route.
6. Enslave. `ip link set eth0 master br0`, then `ip link set eth0 up`.
7. Write `/etc/kvm/network/l2-uplink` containing `br0`, atomically.
8. Address `br0`. On the static path, replay the captured address and route onto `br0` in a single `ip -batch`, and add the nameserver line only if the snapshot shows it was there, rather than re-running `S30eth start`, whose `:55` append would accumulate a duplicate. On the DHCP path, run `/etc/init.d/S30eth start`, which now reads `br0` from `l2-uplink` and launches `udhcpc -i br0` with the existing timers.
9. Write `/etc/kvm/gateway` with the resulting default gateway, so an unpatched `kvm_system` reads it verbatim instead of grepping `ip route` for `eth0`.
10. Re-install the five `S95nanokvm` rules against `br0` and delete the five naming `eth0`.
11. Verify. All three of the checks below must pass.
12. Disarm. Remove `pending.json` and write `last-known-good.json` in one atomic sequence, marking the bridge enabled.
13. Enslave `usb0`, if the profile has a gadget NIC. `presentation.Manager.NIC` reports it, and reports none when no network function is linked into `configs/c.1`, which is probed rather than read off a `/boot` sentinel. Whether that function is `ncm.usb0` or `rndis.usb0` is D10's choice and not this step's: the bridge enslaves whichever is present, then `ip link set usb0 master br0` and `ip link set usb0 up`. This step takes its own smaller snapshot and its own rollback, since its worst case is a host with no network rather than a device with no management plane.

### Verification

Three checks, all required, evaluated before step 12.

`br0` has an IPv4 address, tested with `ip -4 addr show dev br0` rather than a `grep inet` that also matches `inet6`.

The default route is via `br0` and the gateway answers, tested with `ping -I br0 -w 1 <gw>`.

An inbound liveness proof: the server's HTTP listener accepted a request whose local address is the `br0` address at some point since the apply began. If no client was watching, the fallback is a self-connect to `https://<br0-addr>/api/vm/info` from a socket bound to the `br0` address, which proves the listener and the local delivery path rather than the wire, and is recorded as the weaker of the two. Gateway ping alone does not prove the management plane is reachable. The `DROP OUTPUT tcp --sport 8000` rule is standing proof that L3 reachability and service reachability are different properties on this device.

Any failure restores the snapshot, restores `l2-uplink`, restores `/etc/kvm/gateway` and `/etc/resolv.conf`, re-runs `S30eth start`, and returns a structured error naming which of the three checks failed.

## The disable transaction

1. Snapshot, exactly as in enable.
2. Arm the dead-man with a sixty-second deadline.
3. Release `usb0` first, `ip link set usb0 nomaster`, leaving the gadget function itself as the profile has it, since an unbridged `usb0` is the stock state.
4. Kill any running `udhcpc` named by `/run/udhcpc.eth0.pid` and remove the pidfile.
5. Capture `br0`'s IPv4 addresses and default route, then flush `br0` and delete its default route.
6. Release `eth0`, `ip link set eth0 nomaster`, then `ip link set eth0 up`.
7. Remove `/etc/kvm/network/l2-uplink`, so the absent-file fallback returns every reader to `eth0` and the on-disk state is byte-identical to a device that never had the bridge.
8. Address `eth0`, by the same static replay or `S30eth start` split as enable step 8.
9. Rewrite `/etc/kvm/gateway` with the resulting default gateway.
10. Re-install the five firewall rules against `eth0` and delete the five naming `br0`.
11. Verify the same three checks against `eth0` rather than `br0`.
12. Disarm. Remove `pending.json` and write `last-known-good.json` marking the bridge disabled.
13. Tear down the device, `ip link set br0 down` and `ip link delete br0`, after the disarm, so a failed verification restores onto a bridge that still exists.

## Files

| Path | Action |
|---|---|
| LicheeRV-Nano-Build fork, branch `nanokvm-custom`, commit `f74b732` | external, done: `CONFIG_BRIDGE=y`, `BRIDGE_NETFILTER` left unset |
| `kvmapp/system/init.d/S29bridge` | new: `pending.json` check, `br0` creation with `eth0` enslaved |
| `kvmapp/system/init.d/S30eth` | modify: the `IFACE` indirection at eleven sites, the `ip -4` fix at `:59,61` |
| `kvmapp/system/init.d/S95nanokvm` | modify: the ten interface matches in the five rules at `:92-105` |
| `kvmapp/jpg_stream/S95nanokvm` | unchanged, and recorded as the stale duplicate of the same block |
| `server/service/presentation/network/bridge.go` | new: the two transactions |
| `server/service/presentation/network/snapshot.go` | new: `ip -j` capture and restore |
| `server/service/presentation/network/deadman.go` | new: arm, deadline goroutine, disarm |
| `server/service/presentation/network/verify.go` | new: the three checks |
| `server/service/presentation/network/store.go` | new: atomic store over `/etc/kvm/presentation/network/` |
| `server/service/presentation/network/service.go` | new: gin handlers |
| `server/router/network.go` | modify: three routes in the admin group |
| `server/service/vm/ip.go` | modify: one line, `br` prefix maps to `Wired` |
| `server/service/network/dns.go` | modify: one line, the same in `getDNSInterfaceType` |
| `support/sg2002/kvm_system/main/lib/system_state/system_state.cpp` | modify: one `uplink_name()` helper, four call sites |
| `web/src/api/network/bridge.ts` | new |
| `web/src/pages/desktop/menu/settings/network/` | modify: one toggle, one confirmation modal, one state panel |
| `web/src/pages/desktop/menu/settings/device/virtual-devices.tsx` | modify: the NCM/RNDIS selector on the existing Virtual Network control, D10 |
| `web/src/api/virtual-device.ts` | modify: an optional protocol on the update call |
| `web/src/i18n/locales/*.ts` | modify: all 24 |

The apply modal stays, on the same rule the other two programs use. Enabling the bridge moves the address the caller is talking to, so the action can cut the caller's own connection, and it is the kind of thing a modal exists for.

## API

Three routes, joining the existing admin group in `server/router/network.go`, which already carries `CheckToken()` and `RequireRole(authn.RoleAdmin)`. There is no eth0 addressing API today of any kind, so there is nothing to stay compatible with.

```
GET    /api/network/bridge         # state, ports, addresses, last apply outcome
POST   /api/network/bridge         # {"enabled": bool}
POST   /api/network/bridge/revert  # force-restore the snapshot
```

`GET` returns the uplink name, whether `br0` exists, its ports and their states, its address and default route, whether a `pending.json` is armed with its deadline, and the contents of `last-known-good.json`.

`POST` runs the transaction synchronously under the sixty-second deadline and returns `{ "state": "...", "uplink": "...", "checks": {"address": bool, "gateway": bool, "inbound": bool}, "message": "..." }`. The caller losing its connection partway through is the expected failure mode rather than an edge case, so the outcome is durable in `last-known-good.json` before the response is written, and a client that never sees the response reads the result back from `GET`.

`POST /revert` exists for the case where verification passed and the operator still cannot reach the device from somewhere the verification did not test, reached over the `wlan0` AP or a serial console.

## Risks

**R3.1 Bridge creation or enslavement fails on the real device.** Nothing in the tree has ever run `ip link add name br0 type bridge` on this kernel, enslaved a live `eth0` that carries the management address, or observed what DHCP and carrier do once `eth0` is a port rather than the uplink. The config being built in settles only that the code is present in the image. Everything else in Program 3 is blocked on the transcript. Retires when a transcript from a device flashed with the `f74b732` image captures all of `zcat /proc/config.gz | grep -E 'BRIDGE|STP|LLC|BRIDGE_NETFILTER'` showing `CONFIG_BRIDGE=y`, `CONFIG_LLC=y` and `CONFIG_STP=y` with `BRIDGE_NETFILTER` unset, `ip link add name br0 type bridge stp_state 0` succeeding, `ip link set eth0 master br0` succeeding against an `eth0` that was up and addressed beforehand, `bridge link show` listing the port in state `forwarding` within one second of carrier, and `udhcpc -i br0` taking a lease under the existing `-t 10 -T 1` timers.

**R3.2 Management reachability is lost and the device needs physical access.** The DHCP branch at `S30eth:68` has no fallback of any kind, `RESERVE_INET` exists only inside the `/boot/eth.nodhcp` branch, STP can delay carrier past `udhcpc -t 10 -T 1`, the five `-i eth0` rules stop applying to a bridged device, and `S30wifi:116` deletes the default route in AP mode. Retires when the dead-man rollback is proven three ways on hardware: apply with the bridge deliberately misconfigured, for instance with STP left on, and confirm automatic restore within the deadline; hard power-cut the device mid-apply and confirm the `S29bridge` `pending.json` check restores it on the next boot; and confirm that `DROP OUTPUT tcp --sport 8000` still blocks port 8000 from the wire after bridging, shown by a packet capture taken from the attached host rather than by an `iptables -L` listing.

**R3.3 The gadget NIC's L2 identity or the OLED's network state silently changes.** A bridge inherits the numerically lowest port MAC, which can be `usb0`'s deterministic `48:da:35:6e:xx:xx` from `S03usbdev:46-50`, breaking the stable-adapter guarantee the comment at `S03usbdev:42-45` exists to provide; and `ip_address()["eth0"]` returning empty pins `eth_state = 1` at `system_state.cpp:402-407`, so `oled_ui.cpp:143` never selects the wired IP. Retires when `ip link show br0` reports `eth0`'s permanent MAC across ten enslave and release cycles of `usb0`, the attached host's own `ip link` shows an unchanged adapter MAC across the same ten cycles, and the OLED reaches `eth_state == 3` with a wired IP displayed, with the `l2-uplink` indirection and `/etc/kvm/gateway` in place, photographed on real hardware.

## Testing

The pure parts run on an x86 development machine with no device: the snapshot serializer against captured `ip -j` output, the restore planner, the deadline arithmetic, the three verification predicates against fabricated `ip -j` and `ping` results, and the firewall rule rewriter against the exact ten matches at `S95nanokvm:92-105`.

`S30eth` gets a shell-level test in the same shape as Program 1's golden traces: run `start` under a PATH shim in which `ip`, `arping` and `udhcpc` are stubs that append their arguments to a trace file, once with `/etc/kvm/network/l2-uplink` absent and once containing `br0`, across the four starting states `{no eth.nodhcp, eth.nodhcp with a usable line, eth.nodhcp whose first line fails arping, eth.nodhcp with no usable line}`. The absent-file traces must be byte-identical to the traces the unmodified script produces, so the fallback claim in D2 is checkable rather than asserted. The `br0` traces must differ from those in exactly the interface argument. A separate case asserts that `:59` and `:61` no longer accept a device carrying only an IPv6 link-local, by feeding the stub an `ip -4` output with no `inet` line.

The rest needs hardware. The bridge itself is tested by the R3.1 transcript. The rollback is tested by the three R3.2 exercises. The MAC pin and the OLED are tested by the ten cycles and the photograph in R3.3. Beyond those: the router issues two distinct DHCP leases, one to the NanoKVM's pinned MAC and one to the attached host's own adapter MAC, confirmed from the router's lease table rather than from either endpoint; a reboot with the bridge enabled comes up addressed with no live handoff, since `S29bridge` created `br0` before `S30eth` ran; and `GET /api/vm/info` lists `br0` in `ips[]` with type `Wired` while the About panel renders it.

No CI workflow in this repo runs `go test` or `go vet`, so the Go tests run locally and in review until that changes. `make web` does run `tsc && vite build`, so TypeScript errors fail the build.

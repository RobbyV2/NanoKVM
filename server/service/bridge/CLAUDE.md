# bridge

Puts `eth0` and the USB gadget NIC into one Layer-2 bridge, `br0`, so the router sees the
NanoKVM and the controlled host as two devices with two MACs and two DHCP leases.
Design: `docs/superpowers/specs/2026-08-22-ethernet-bridge-design.md`.

Enabling the bridge moves the address the caller is talking to. Every rule below exists
because the failure mode of this package is a device that needs physical access.

The on-disk state lives at `/etc/kvm/presentation/network/` even though the Go moved to
`service/bridge`. `kvmapp/system/init.d/S29bridge` hardcodes that path, so renaming the
directory means editing the shell script and shipping both in the same release.

## The dead-man contract, and the ordering that makes it work

Snapshot first, then arm, then mutate. `Manager.begin` writes `snapshot.json` and fsyncs
it before `arm` writes `pending.json`, and `arm` returns only once the marker is durable
and the watchdog goroutine is running. There is no window in which the device has been
changed and nothing is watching, and no window in which a marker points at a snapshot
that does not exist yet. Reversing those two writes turns a power cut into a device that
knows it was mid-apply and cannot say what it was before.

The marker is cleared last on every path. `restoreFrom` restores, records the outcome,
and only then commits; an interruption anywhere inside leaves a marker the boot-time
check acts on again. `commit` writes `last-known-good.json` before removing the marker
and before the HTTP response is written, because the caller losing its connection partway
through is the expected outcome rather than an edge case: a client that never sees the
response reads the result back from `GET`.

The watchdog runs on `context.Background()`, not the transaction's context. The
transaction's is very likely the one that just expired, and a restore that inherits a
cancelled context restores nothing.

`deadman.take` and `disarm` are a compare-and-swap on the same flag. If the deadline fires
first, the watchdog has already restored and recorded its own outcome, and `commit` must
not overwrite that with a success that was undone.

## Two halves of the boot path, and why the record is a rename

`RecoverPending` is the server's half. A marker still armed when the server starts means
this process died inside an apply without a reboot, since `S29bridge` consumes one at
every boot. It restores unconditionally without consulting the deadline: the process that
armed it never disarmed, and the deadline is measured against a clock this device probably
did not keep across the cut.

`S29bridge` is the boot half, and it exists because it has to run before `S30eth` addresses
anything, which is before the server exists. So the script performs the restore itself.
`new_app_init()` refreshes it into `/etc/init.d` immediately before `S30eth`; keeping only
the seed under `/kvmapp` makes both recovery and persistent bridge setup unreachable.
It moves `pending.json` to `recovered.json` rather than deleting it. The rename is the
whole write: it cannot leave a partial file, and it clears the marker in the same
operation, so a second boot cannot restore twice. Deleting instead would leave the device
silently back on `eth0` with no record of why, indistinguishable from a bridge that was
never enabled. `Manager.adoptRecovered` turns that record into the `lastApply` that `GET`
reports, writing the outcome before clearing the record so an interruption between them
leaves it to be adopted again rather than lost.

## STP is off, not merely fast

`ip link add name br0 type bridge stp_state 0`. With STP enabled a port goes through
listening and learning before it forwards, roughly thirty seconds, and a bridge does not
report `IFF_RUNNING` until a port forwards. `S30eth` runs `udhcpc -i <uplink> -t 10 -T 1`,
which gives up after ten one-second tries, and its DHCP branch has no fallback of any
kind: `RESERVE_INET` exists only inside the `/boot/eth.nodhcp` branch. So a bridge with
STP on takes a lease exactly never, and the device comes up with no IPv4 and nothing that
will retry.

`forward_delay 0` is set alongside and is redundant while STP is off. It is there so that
nobody later reads the redundancy as the load-bearing setting. `stp_state 0` is the
load-bearing setting.

There is no loop risk to trade against: this is a two-port bridge whose second port is a
USB gadget.

## The bridge MAC is pinned to eth0's

`permanentMAC` reads `/sys/class/net/eth0/address` before anything is enslaved, and
`SetMAC` writes it onto `br0`. Setting an address explicitly marks the bridge address
static, so `br_stp_recalculate_bridge_id()` stops re-electing the numerically lowest port
address every time a port is added or removed.

Without the pin, enslaving `usb0` at its deterministic `48:da:35:6e:xx:xx` can win that
election, the bridge's L2 identity changes under a live DHCP lease, and the reservation
the operator configured against the old address stops matching. That is the stable-adapter
guarantee the comment at `S03usbdev:42-45` exists to provide, broken from the other side.
Read the MAC before enslaving: once `eth0` is a port, the value that matters is still
`eth0`'s own and not whatever the bridge settled on.

## wlan0 is refused by the enslavable set

`enslavable` in `config.go` is a closed map holding `eth0` and `usb0`. `wlan0` is absent by
construction, so the rule is enforced by the data rather than by a check every call site
has to remember, and `checkEnslavable` reports it as its own error because that rejection
has a cause worth surfacing.

`wlan0` is the out-of-band recovery path. It is how an operator reaches a device whose
wired management plane the bridge just broke, and `POST /api/network/bridge/revert` exists
to be reached over it. Bridging it away is the one move that makes the recovery path
depend on the thing being recovered from. `hostapd` plus bridging is separately its own
category of problem.

## The inbound gate proves the listener answers, not that the wire forwards

Verification is three gates and all three must pass: an IPv4 address on the uplink, read
with `ip -4` rather than a `grep inet` that also matches `inet6`; a default route through
the uplink whose gateway answers a `ping -I <uplink>`; and inbound liveness.

Gate three is the one worth understanding. The strong form is `Observed`: the HTTP
listener accepted a request whose local address was the uplink address at or after the
apply began, which means a real client completed a round trip over the wire. With no
client watching, nothing is recorded and it falls through to `SelfConnect`, which dials
the listener from a socket bound to the uplink address. That proves the listener and the
local delivery path. It does not prove the wire forwards, and it is recorded as
`InboundWeak` so an operator reading the result knows which of the two they got. Do not
collapse the distinction.

Gate two alone would not do. `S95nanokvm` installs `DROP OUTPUT tcp --sport 8000` to keep
the raw stream port off the wire, which is standing proof that L3 reachability and service
reachability are different properties on this device. A gateway that answers ping says
nothing about whether anyone can reach the management plane.

`SelfConnect` skips TLS verification. The device serves a certificate it generated itself,
so verification would fail on every device and prove nothing about reachability either
way. Any HTTP status counts: a 401 is still the listener answering.

## The gadget port does not stay put

A presentation apply unbinds the UDC and binds it again, and the kernel destroys and
recreates `usb0` with no memory of `br0`, so enable's step 13 holds only until the next
profile change. `S29bridge` builds `br0` from scratch at every boot, so a script that
enslaves `eth0` alone comes up one-ported. Both halves lose the second port silently.

`bridge.Gadget` gains `OnRebind` for the first half: `New` hands the presentation manager
`ReattachGadget`, and the manager calls it after every apply, failed ones included, since a
failed apply rebound the UDC too. `S29bridge` covers the second, enslaving `usb0` inside
`create` after `eth0` and before the uplink file is written.

Neither path may fail on an absent `usb0`. It exists only when the active profile has a
network function linked into `configs/c.1`, and bridge-enabled-with-no-NIC is legitimate:
`NIC` reports empty and `enslaveGadget` does nothing, and the script's `enslave_gadget`
returns 0 on every failure it can meet, because a non-zero return there drops `start` into
`teardown` and costs the management address for the sake of the attached host's network.
`ReattachGadget` reads the live link list first and returns without a `br0`, which is every
device that has never enabled the bridge.

## Smaller things that will bite

`UdhcpcPidPath` stays the literal `/run/udhcpc.eth0.pid` under every uplink. It names
`S30eth`'s one DHCP client rather than an interface, and deriving it from the uplink would
leave `stop` unable to find a client that `start` launched under the other name. A live
enable passes through exactly that state.

`/etc/resolv.conf` is captured and restored because `S30eth:55` appends its nameserver line
rather than replacing it, so re-running the script accumulates duplicates. That is why the
static path replays the captured address itself instead of calling `S30eth start`.

`/etc/kvm/gateway` is written on every apply because `system_state.cpp` reads it verbatim
in preference to its own `ip route | grep eth0`. An OTA can ship a new server without a new
`kvm_system`, and that file is what keeps the OLED showing a gateway on such a device.

`usb0` is enslaved after the dead-man is disarmed. Its worst case is an attached host with
no network, not a device with no management plane, so it takes its own smaller snapshot and
its own rollback rather than sitting inside the one that holds the management address.

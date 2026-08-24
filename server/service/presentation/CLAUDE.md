# presentation

Owns the USB gadget under `/sys/kernel/config/usb_gadget/g0`. Compiles a declarative
`Profile` into an ordered `Plan` of configfs operations and applies it under one lock.
Design: `docs/superpowers/specs/2026-08-22-presentation-manager-design.md`.

## The package is add-only, and that is not a style choice

`f_hid` allocates the `/dev/hidgN` minor from an `ida` at `mkdir functions/hid.GSn` time,
in creation order. `service/hid/hid.go:29-32` hardcodes hidg0 as the keyboard, hidg1 as
the relative mouse and hidg2 as the absolute pointer. Nothing at runtime reads which
function owns which minor, so if `functions/hid.GS0` is ever removed and recreated the
numbering shifts and the server writes mouse reports into the keyboard endpoint. There
is no error and no log line. The attached host sees garbage keystrokes.

So a transaction here never `rmdir`s `functions/hid.*` and never `rmdir g0`.
`validateRemove` in `ops_configfs.go` restricts `Ops.Remove` to symlinks under
`configs/c.1/` plus the `os_desc/c.1` link and nothing else, `Profile.Validate` rejects
any profile whose HID functions are not exactly GS0, GS1, GS2 in that order, and
`apply_test.go` asserts that standard, then hid-only, then standard emits zero ops
touching `functions/hid.*`. Keep all three. Do not add a HID teardown path because a
rebuild would be tidier.

## Dropped net and disk functions do lose their directories, and that is not tidiness

The rule above is about `hid` and only about `hid`. Unlinking any function from a config
does not destroy it, and for `f_ncm` and `f_rndis` that is a live bug: both call
`gether_setup` at mkdir, which registers a netdev under gether's `"usb%d"` and holds it
until `rmdir`. An unlinked-but-present `rndis.usb0` therefore goes on owning the name
`usb0`, and the `ncm.usb0` created to replace it is handed `usb1` by the kernel's
first-free allocation. Everything that then asks for the gadget NIC by name - the bridge
port, `S30rndis`'s addressing, `udhcpd`'s `interface` line - binds to a netdev with no
carrier and no function behind it, and only a reboot clears it. This was observed on
hardware: `br0` carrying a dead `usb0` while the live NCM link sat unbridged on `usb1`.

`unlinkStale` therefore releases the directory of every dropped function whose kind is in
`releasableKinds` - `ncm`, `rndis` and `mass_storage` - between the unlink and the plan's
mkdir, which is the only window where the name is free and `rmdir` is not `-EBUSY`. `hid`
is excluded for the ida reason above. `uvc` and `uac2` are excluded because the media
manager owns their nested descriptor groups and their `/dev/video` and ALSA indices, and
because `Plan.Outcome` lists every media function as removed on every apply since it
relinks them.

Because the netdev name is now allocated rather than assumed, `Manager.NIC` reads
`configs/c.1/<net-fn>/ifname` and returns the kernel's own answer. `GadgetNIC` is only the
fallback for a kernel that does not publish the attribute.

## The gadget is reconciled against the profile at startup

`S03usbdev` rebuilds the stock three-HID gadget from scratch on every boot and knows
nothing about the profile store, while `Migrate` returns early the moment an active
profile exists. Between them, a layout the operator chose and an apply that rolled back
leave the store promising one gadget and the kernel presenting another - permanently,
silently, and with the server writing HID reports shaped for the promise into endpoints
that belong to the other one. That is how a device ended up showing HID devices in
Windows that controlled nothing: a stored `report_length: 9` report-ID composite written
into what was really the stock eight-byte boot keyboard.

`Manager.ReconcileGadget` runs from `GetManager` after `Migrate`, compares
`compareLayout(activeProfile, liveFunctions(ops))` and reapplies the profile when they
differ. It is what makes a collapsed HID layout survive a reboot at all. A reconcile that
cannot land the profile is logged and never fatal: every rung of `applyPlan`'s ladder ends
in a bind, so the controller is left bound whatever happens.

**The routes follow the gadget, never the store.** `Manager.hidRoutes` builds
`[]HIDRoute` from the live functions read back out of configfs, and consults the stored
profile only to put role names to a descriptor this package did not compose. Roles are
recovered by composing all fifteen layouts `hidLayoutFunctions` can build and comparing
`report_desc` bytes, which is exact and needs no descriptor parser; the node number comes
from `f_hid`'s own `dev` attribute rather than from link order. This is the safety
property: a mismatched profile still leaves the operator a working keyboard.

`Snapshot.Diverged` re-derives the same comparison on every read and is stored nowhere. A
marker file would go stale the next time `S03usbdev` rewrote the gadget under it. A failed
apply folds the same derivation into its error as `ErrDiverged`, because `applyPlan`'s
ladder reverts the store along with the gadget on every rung that completes and the only
case left is the one where none of them did.

The same rule is why `capability.go` never probes `hid.*`: a scratch `mkdir functions/hid.probe`
consumes a minor and shifts the numbering for everything created after it.

## The golden traces are the acceptance criteria

`testdata/traces/*.trace` holds 20 recorded runs of the shell scripts this package
replaces. `compile_test.go` compiles the equivalent profile for each and compares the
whole op sequence: order, exact bytes including trailing newlines, and symlink target
strings. A change that alters what `Compile` emits and does not update a trace is a
change to the gadget the attached host enumerates.

The traces are recorded from the scripts rather than written by hand, because the
scripts have no `set -e` and several of their writes already fail invisibly on a
shipping device. `echo e0 > functions/rndis.usb0/class` is parsed by `kstrtou8(page, 0, ...)`,
base 0 rejects unprefixed hex, and the value happens to equal the RNDIS IAD default, so
the failure leaves no trace anywhere. Go that faithfully performs every write the script
intends produces a different gadget from the one the script produces. Comparing against
observed behaviour rather than against a reading of the script is the only way to catch
that.

Regenerate with the command in the header of `testdata/gen_traces.sh`:

```
docker run --rm -v $PWD:/repo alpine:3.22 sh /repo/server/service/presentation/testdata/gen_traces.sh /repo
```

It runs the unmodified logic of `kvmapp/system/init.d/S03usbdev` and `S03usbhid` under
BusyBox ash against a sandbox tree, relocating only the four absolute roots the scripts
reach outside the gadget. `mkdir`, `ln`, `cat`, `ls` and `echo` are replaced by shell
functions rather than PATH stubs, because `echo` is an ash builtin and a PATH stub can
never intercept it. Regenerate only when the init scripts change. Regenerating to make a
failing test pass throws away the reason the test exists.

## Compile is pure, Ops is the boundary

`compile.go` imports nothing outside stdlib, touches no filesystem, and is the only place
that decides what the gadget should be. `ops.go` is the only place a syscall happens.
Keep both true: an `os.WriteFile("/sys/kernel/config/...")` anywhere else in the package
removes the split.

There is no privileged helper process today, and the reasoning is in the spec: the server
runs as uid 0 out of BusyBox `rcS`, holds `/dev/hidg0-2` open for its process lifetime,
and two admin endpoints already hand out a root shell, so a helper would guard a door in
a wall with a hole in it. What is built now is the part that makes the split cheap later.
`Plan` is `[]Op` of `{Kind, Path, Target, Data}` and marshals to JSON, so a future helper
needs a transport and nothing else: the server hands over an already-compiled,
already-validated plan and the helper executes it, never parsing a profile and never
trusting user JSON. Path validation in `compile.go` is the check that helper would need
anyway. Revisit when `service/vm/terminal.go` and `/api/vm/script/run` are gone.

## Ordering rules the kernel enforces

The symlink comes last for `hid`, `ncm` and `rndis`. Every `f_hid`, `f_ncm` and `f_rndis`
option store checks `opts->refcnt` and returns `-EBUSY` once the function is linked into
a config, so `report_desc`, `report_length`, `protocol`, `subclass`, `dev_addr`,
`host_addr` and `class` are writable only before the link. Reordering a HID link does not
error: it enumerates a 64-byte device carrying the kernel's default report descriptor.

`mass_storage` is the exception and `compiler.storage` links first on purpose. The LUN
attributes have no refcnt check, and `S03usbdev:135` relies on that. `lun.0/ro` and
`lun.0/cdrom` still return `-EBUSY` while `lun.0/file` is open, which is why
`setLUN` closes the file first.

`compiler.uvc` links the SuperSpeed control and streaming class groups as well as the
full- and high-speed ones the init script wrote, even though the device's dwc2 is
high-speed only. A 5.10 `f_uvc` guards its SuperSpeed descriptor copy behind
`gadget_is_superspeed` and ignores them; from 5.15 that guard is gone and
`uvc_function_bind` fails with a bare `-ENODEV` and no log line when the ss class links
are missing. Two extra symlinks are the whole cost.

The link order in `configs/c.1` fixes `bInterfaceNumber` assignment and therefore
host-side driver binding, so `Profile.Functions` order is the link order and is
reproduced exactly: net function, then hid.GS0, GS1, GS2, then mass_storage.disk0.

## The UDC is one pointer, and two things want it

`udc->driver` is a single pointer, so this gadget and a `usb-proxy` passthrough session
cannot both hold the controller. `SurrenderUDC` unbinds and records a **loan**; every
mutator refuses with `ErrUDCLoaned` while one stands. The check sits in `withGadgetLock`,
which covers `Apply`, `ApplyProfile`, `SetMode`, `SetMediaSlots`, `SetLUN`, `Rebind` and
`ResetPHY`, and is repeated at the four entry points that test `m.transient` directly:
`CreateFunctionFS`, `StartFunctionFS`, `RecoverFunctionFS` and `SurrenderUDC` itself. A
transient and a loan cannot coexist, since each refuses the other, which is why
`StopFunctionFS` does not check.

Three properties are load-bearing. The loan is **memory only** — nothing is written to
disk, so no stale file can outlive a reboot and wedge the gadget. It is reconciled
against the kernel rather than against the record: `loanHeld` reads the gadget's `UDC`
attribute and drops the loan when it is non-empty, because the borrower holds the
controller only while this gadget is unbound, so a loan standing against a bound gadget
is stale by construction. And `ReclaimUDC` clears it **before** the bind and
unconditionally — a failed bind still leaves the controller free, so re-taking the loan
there would refuse every mutator until the next reboot, which is worse than the rebind it
was guarding.

Refuse before you suspend. `observer.Suspend()` is undone only by the observer's
`Applied`, which the refusal paths never reach, so a check placed after the suspend
strands the media pipeline on a surrender that never happened. `StartFunctionFS`,
`RecoverFunctionFS` and `SurrenderUDC` all take `m.mu`, refuse, and only then suspend.

## Every rebind destroys the gadget NIC

Mutating the gadget unbinds the UDC and binds it again, and the kernel destroys and
recreates `usb0` with no memory of the bridge it was a port of. `OnRebind` is how the
bridge learns: it registers one callback, and `notifyRebound` fires it from
`refreshObserver` and `notifyObserver`, the two funnels every mutation path already ends
in on both success and failure. That is why `Apply`, `ApplyProfile`, `SetMediaSlots`,
`Start`, `Stop`, `RecoverFunctionFS`, `SetLUN`, `ReclaimUDC`, `Rebind`, `ResetPHY` and
`SetMode` are all covered by two call sites.

The hook is deliberately over-eager: it also fires on paths where nothing was rebound,
because re-enslaving a port that is already a port is a no-op and missing a real rebind
is not. Nothing in this package knows a bridge can exist — the dependency points bridge
into presentation and never back — and the boot half of the same durability lives in
`S29bridge`, which enslaves `usb0` when it builds `br0`.

## bcdDevice is the mode marker

`S03usbdev` never wrote `bcdDevice` in any revision. `service/hid/status.go` worked anyway
because the gadget core's `get_default_bcdDevice()` computes
`bin2bcd(VERSION) << 8 | bin2bcd(PATCHLEVEL)`, which is `0x0510` on Linux 5.10, and normal
mode was identified by that value. `S03usbhid` writes `0x0623`, and that is still the
HID-only marker.

`Manager.Mode` now resolves in three tiers: the active profile first, an exact match
against `0x0510` or `0x0623` second, and any `0x05xx` third. The tolerant tier exists
because the old marker was an accident of the kernel version. A vendor bump to 5.15
yields `0x050f` or `0x0515`, and without the tolerant tier `GetMode` returns
`invalid mode flag`, which breaks `/api/hid/mode`, breaks `/api/storage/image/mounted`
(`storage/image.go` hard-fails the endpoint on a `GetMode` error) and makes `SetHidMode`
miss its already-in-that-mode short circuit and reboot the device. Do not tighten it to
an exact match.

## The capability table, and why six IN is silicon

`staticV1` is a plain data literal and the only table this package ships. Its six IN
endpoints started as what the shipping scripts demonstrably achieve at full flag
expansion. They have since been read off the hardware and they are right, but for a
reason worth writing down, because the obvious workaround does not exist.

`/sys/class/udc/*` does not carry the budget, but `CONFIG_DEBUG_FS=y` on the device and
`S01fs` mounts it, so `/sys/kernel/debug/usb/4340000.usb/` does. On the SG2002:

```
num_dev_ep       : 7        GHWCFG2 = 0x228f5c52
total_fifo_size  : 3072      GHWCFG3 = 0x0c0004e8
en_multiple_tx_fifo : 1      GHWCFG4 = 0xda00ba30 -> NumDevInEps = 6
```

`dwc2_hsotg_tx_fifo_count` returns `hw_params.num_dev_in_eps` in dedicated-FIFO mode, and
`dwc2_hsotg_ep_enable` refuses an IN endpoint it cannot give a unique TX FIFO to. So the
ceiling is six IN endpoints besides ep0, whatever the device tree says. Widening
`g-tx-fifo-size` past six entries does not help and is actively harmful: reading
`DPTXFSIZ1..8` through `devmem` on a running device gives

```
0x104 0x03000238  0x108 0x02000538  0x10c 0x02000738  0x110 0x01800938
0x114 0x00800AB8  0x118 0x00800B38  0x11c 0x00800AB8  0x120 0x00800B38
```

where `0x11c` and `0x120` return exactly what `0x114` and `0x118` hold, and the pattern
repeats again from `0x124`. Only six of those registers are implemented; a seventh entry
in the device tree would land on FIFO 5's configuration. The FIFO *depths* are a real
allocation - 536 rx + 32 np-tx + 2432 tx = 3000 of 3072 words - but there is nothing to
spend the remaining 72 words on.

Availability and per-function attributes can be probed, and `LoadCapabilities` merges
probe results for those only, never for the budget.

The predecessor `staticV0` is gone. It lacked the `uvc` and `uac2` entries and every
`INPackets` value, so `supportsMedia` rejects a table of that shape on disk and
`LoadCapabilities` could never return it. Tests that compiled against it were exercising
a table no device runs, and in particular were seating no FIFOs at all for `ncm`,
`rndis` and `mass_storage`, whose IN packet sizes live only in `INPackets`. Every test
compiles against `staticV1`. Do not reintroduce a second table that ships nowhere.

`FunctionCaps.Attributes` is how a kernel option a profile may count on reaches the
allocator. `UVCAttrInterruptEP` is the one that changes an endpoint cost: `f_uvc` autoconfigures
a control interrupt IN endpoint and never queues a request to it, since every UVC event
reaches userspace through `v4l2_event_queue`, so the endpoint costs a dedicated FIFO to
advertise a channel nothing writes to. The kernel fork adds `enable_interrupt_ep` to the
uvc function group; `probeAvailability` stats it on the scratch gadget, and a profile that
sets `Video.InterruptEndpoint` to false is refused outright on a kernel without it rather
than silently under-counting and failing at bind.

`UVCAttrFunctionName` and `UAC2AttrFunctionName` are the other two, and they are a
different `function_name` from the read-only sysfs attribute `service/media/resolver.go`
maps a `/dev/videoN` or `hw:N,0` back through. The writable configfs one sets the string a
target host displays for that camera or microphone, which is what stops eight browser
sources from all reading "UVC Camera". `Video.HostName` and `Audio.HostName` are nilable
for the same reason `InterruptEndpoint` is: nil means the attribute is not written at all,
so every profile stored before this existed keeps whatever name the kernel picked and an
upgrade renames nobody's camera. `supportsMedia` requires the two keys to be *present*
rather than true, so a `probe-v1` table written before they were probed is discarded and
re-probed instead of reading as "this kernel cannot name media devices".

`InFIFOWords` is the six dedicated dwc2 IN FIFOs in words, and `SeatFIFOs` assigns the
smallest FIFO that holds each IN packet, so a plan is refused when a packet fits no
remaining FIFO. `inPackets` prefers the profile's own payload where one exists (HID
report length, UVC `streaming_maxpacket`, the UAC2 playback packet, FunctionFS endpoint
sizes) and falls back to the table's `INPackets` otherwise.

Consequences worth knowing before you touch it. Both built-ins compile against `staticV1`
with zero headroom by construction, so neither can be refused by its own table, and
`capability_test.go` pins that. The allocator only ever refuses a plan and names the
function it refused for; it never sizes one. `Source` is carried into every error message
so a rejection reads `rejected by capability table static-v1` and an operator can tell a
measured refusal from a guessed one. If a real endpoint budget is ever measured, write it
to `/etc/kvm/presentation/capability.json` rather than editing the literal, which is what
the precedence order in `LoadCapabilities` is for; a table written there is ignored unless
it carries the media functions and a FIFO map.

## The HID layout

Three roles - keyboard, relative mouse, absolute pointer - are distributed over one, two
or three `hid.GS*` functions, and a function carrying more than one role separates them
with Report IDs. That is the only lever that gets camera, microphone, NIC and a full HID
set inside six IN endpoints:

```
uvc.cam0 2 + uac2.mic0 1 + ncm.usb0 2 = 5 IN, leaving one for HID
```

Dropping the control interrupt endpoint takes `uvc` to 1 and leaves room for two HID
interfaces, which is what keeps the boot-protocol keyboard on an interface of its own.
`hidLayoutFunctions` clears `protocol` and `subclass` on any interface carrying more than
one role, because `bInterfaceProtocol` 1 and the boot subclass promise a boot report a
report-ID composite cannot produce.

Two properties are load-bearing. A one-role group returns the byte-pinned descriptor
unchanged, so the default layout compiles to exactly what the golden traces record.
And the instances stay a **prefix** of GS0, GS1, GS2 in order, for the reason at the top
of this file: `f_hid` hands out `/dev/hidgN` in mkdir order, so a layout that skipped GS1
would put the pointer on the mouse's minor.

`HIDRoutes` is the other half. `service/hid` no longer hardcodes hidg0/1/2: the manager
pushes the active profile's routes through the optional `HIDRouter` interface inside the
`withHIDQuiesced` bracket, before the writers reopen, so a role that has moved node or
gained a Report ID prefix is writing to the right place by the time the gadget is back. A
nil route table means no layout has been pushed and every role keeps its historical node,
which is what makes an unwired `Hid` behave exactly as it did before roles existed.

## Store and sentinels

`/etc/kvm/presentation/`, directory 0755, every file 0600, written through the shared
atomic helper. Built-in profiles are code: `standardProfile()` and `hidOnlyProfile()`
return `Profile` values with byte-pinned report descriptors, written to disk for
inspectability and always reconstructed from code on load, so a corrupt `standard.json`
cannot brick USB and an OTA can fix a built-in.

The last-known-good profile is loaded and compiled before every unbind. Profile files are
saved only after the new gadget binds and verifies, so updating the active profile in
place cannot overwrite the rollback copy first. Any later failure restores the prior
profile, sentinels and markers. If rollback also fails, the original and rollback errors
are both returned and the manager makes one final bind attempt so the UDC is not left
empty merely because recovery was incomplete.

`/boot/usb.ncm`, `/boot/usb.rndis0` and `/boot/usb.disk0` are write-through mirrors, not
inputs. Every apply rewrites them to match the active profile. They exist because
`system_init.cpp` unconditionally copies the kvmapp `S03usbdev` back over `/etc/init.d/`
on every app update, so an OTA or a downgrade can put the shell script back in charge of
the boot path, and it must boot the user's gadget rather than a stale one. Removing them
is a separate change, gated on the init script no longer being the boot-time configurator.

## Profile packages

The admin API stores custom profiles as JSON and keeps built-ins read-only in code. It accepts
incompatible profiles for inspection and export; only preview and apply run the endpoint
allocator. Package imports are ZIP archives with a strict `manifest.json`, SHA-256 references,
and bounded descriptor assets. Device, configuration, BOS, string, and HID report descriptors
are validated and preserved. The ConfigFS compiler still applies only fields and functions it
models, so preserved assets are not evidence that a profile is descriptor-compatible.

## The kernel tests

`//go:build linux && kernelint` marks the tests that mutate a real kernel rather than a
recorder. Darwin is excluded by the `linux` term, so `go test ./...` on a laptop never
compiles them and the normal suite stays green with no `t.Skip` anywhere. Inside the tag
the helpers in `service/kernelint` fail rather than skip. A silent skip is how the fake
gadget in `passthrough` kept the FunctionFS ordering defect green for weeks, and a
harness built to catch that defect must not reintroduce the mechanism that hid it.

Two tiers, spelled out in the test names because CI splits on them.
`TestKernelTier1*` needs a private network namespace and `vhci_hcd`, both of which an
ubuntu-latest runner has, and the **Kernel tier 1** job in `test.yml` runs them on every
push. `TestKernelTier2*` needs a bound UDC, which means `dummy_hcd`, and no distro ships
it: `CONFIG_USB_DUMMY_HCD` is not set on Ubuntu and the module is in neither
`linux-modules` nor `linux-modules-extra`. `scripts/kernelint.sh` builds it out of tree
against `linux-headers` in about seven seconds and boots a stock 6.8 kernel under QEMU,
so tier 2 is `make kernelint-tier2` and takes roughly a minute end to end with no state
carried between runs. GitHub's `-azure` kernel is built with `CONFIG_USB_GADGET` unset,
so tier 2 cannot run on a hosted runner at all and the workflow says so rather than
shipping a job that passes by skipping.

What tier 2 buys that the twenty golden traces do not: the traces assert the compiler
still emits the recorded byte sequence, and `kernel_tier2_test.go` asserts the kernel
still accepts it. `dummy_udc.0` rejects writes a recorder accepts, `f_hid` hands back the
`/dev/hidgN` minors that `hid/hid.go` hardcodes, and `hid-generic` on the host side of
`dummy_hcd` parses the three report descriptors `hidFunctions` ships. The FunctionFS
ordering contract is asserted from `service/functionfs` and `service/passthrough`, since
that is where the two halves live.

### Three things the kernel said that no fake had

`f_hid` takes its `opts->refcnt` at **link** time, not at bind time, and every
`F_HID_OPT` store returns `-EBUSY` while that refcnt is held. `f_ncm`, `f_rndis`, `f_uvc`
and `f_uac2` guard their option stores the same way; the `mass_storage` LUN attributes
are the exception two sections up. `unlinkStale` keeps a link the incoming plan also
carries, so the refcnt is never released, and `S03usbdev:99,114,129` links all three HID
functions at boot, so this is the state every device is in when the server starts. The
*shows* are not guarded, and the script writes exactly the values the built-in profiles
carry, so `dropRedundantWrites` reads each write's attribute back through the gadget and
drops the op when it already holds those bytes. That is what makes the first apply after
boot possible at all, and it is why a second apply of the same profile emits no function
writes.

A write that genuinely differs cannot be dropped, and until `Reconcile` existed it was
left in the plan for the kernel to refuse: applying any profile that changed an
attribute failed with `-EBUSY`, the rollback failed for the same reason, and so did the
hid-only rung below it, which left `c.1` holding three HID links and `functions/`
holding everything the attempt had built. What releases the refcnt is dropping the
`configs/c.1/<name>` symlink, and that is not the removal `R1.1` forbids:
`hidg_alloc_inst` takes the `/dev/hidgN` minor from an `ida` at **mkdir** and
`hidg_free_inst` returns it at **rmdir**, while `hidg_alloc` and `hidg_free` move the
refcnt on **link** and **unlink**. `TestKernelTier2UnlinkKeepsHIDMinors` runs that cycle
against the kernel and the minors do not move.

So the unlink belongs in the plan, and `Reconcile(current Snapshot, plan Plan) Plan` in
`compile.go` puts it there: it is pure, it takes the linkage that is actually up, and it
inserts `unlink configs/c.1/<name>` in front of the first op the refcnt would refuse.
`Compile` stays a plan for a virgin gadget and the twenty traces stay byte-identical.
Every unlink it inserts is paired with the link the same plan already ends that function
with, so a failure between the two is repaired by a rollback rung that links the same
functions, and `applyPlan` and `restore` both run it so the whole ladder can write
attributes. `mass_storage` is excluded because its link precedes its LUN attributes,
which carry no refcnt check; a media function is excluded because `unlinkStale` has
already removed it. A function whose attributes all came back redundant keeps its link,
so a partial relink appends the reconciled functions to the config's `func_list` and the
interface numbers move. Nothing on this side reads them, and every apply already unbinds
and rebinds, so the host re-enumerates either way.

`SetHidMode` stages the mode on disk because the init script is still the boot-time
configurator and `system_init.cpp` restores its `kvmapp` copy on every update. The reboot
next to it is justified in `service/hid/status.go` by the `-EBUSY` on `report_desc`, and
that justification no longer holds; whether the reboot itself is still wanted is a
question for that package, not this one.

`applyPlan` and `restore` unbind through `unbindIfBound`, because an empty `UDC` write is
`unregister_gadget` and returns `ENODEV` unless the gadget is bound at that moment. Every
rollback rung reaches its unbind with the transaction's own unbind already done, which is
why a failed apply used to report `rollback to standard: unbind: write UDC: no such
device` on top of the failure that started it. On a device `S03usbdev` binds before the
server starts, so nothing but a rollback ever reached it.

The fakes in `apply_test.go` accept a write the kernel refuses, and the golden traces
record a script that only ever runs against a fresh gadget, so neither tier of the old
suite could see either defect. `TestKernelTier2ApplyOverBootLinkedHID` and
`TestKernelTier2ApplyBindsAnUnboundGadget` are what hold them now: `bootLinkedHID` builds
the linkage `S03usbdev` leaves and asserts the attributes read back through it, and both
tests reproduce the kernel's own error text when either fix is reverted.
`kernelint.BootstrapGadget` still links a net function rather than HID, not because HID
would fail any more but because the harness is shared with `functionfs` and `passthrough`
and a `hid.*` there costs a `/dev/hidgN` minor no test in those packages asserts on.

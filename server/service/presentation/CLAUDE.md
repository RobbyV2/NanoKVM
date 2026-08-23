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

So a transaction here is bind, unbind, symlink add and symlink remove. It is never
`rmdir functions/*` and never `rmdir g0`. `validateRemove` in `ops_configfs.go` restricts
`Ops.Remove` to symlinks under `configs/c.1/` plus the `os_desc/c.1` link and nothing
else, `Profile.Validate` rejects any profile whose HID
functions are not exactly GS0, GS1, GS2 in that order, and `apply_test.go` asserts that
standard, then hid-only, then standard emits zero ops touching `functions/hid.*`. Keep
all three. Do not add a teardown path because a rebuild would be tidier.

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

The link order in `configs/c.1` fixes `bInterfaceNumber` assignment and therefore
host-side driver binding, so `Profile.Functions` order is the link order and is
reproduced exactly: net function, then hid.GS0, GS1, GS2, then mass_storage.disk0.

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

## The capability table is not a hardware measurement

`staticV1` is a plain data literal and the only table this package ships. Its six IN and
five OUT endpoints are what the shipping scripts demonstrably achieve at full flag
expansion, not a probe result. The real `num_dev_ep` lives in the dwc2 `GHWCFG` registers
and is not readable from `/sys/class/udc/*`, so the budget cannot be probed at runtime at
all. Availability can be probed, and `LoadCapabilities` merges probe results for
availability only, never for the budget.

The predecessor `staticV0` is gone. It lacked the `uvc` and `uac2` entries and every
`INPackets` value, so `supportsMedia` rejects a table of that shape on disk and
`LoadCapabilities` could never return it. Tests that compiled against it were exercising
a table no device runs, and in particular were seating no FIFOs at all for `ncm`,
`rndis` and `mass_storage`, whose IN packet sizes live only in `INPackets`. Every test
compiles against `staticV1`. Do not reintroduce a second table that ships nowhere.

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

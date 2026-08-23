# USB presentation manager

Status: design only. No code written, nothing has run on hardware.
Date: 2026-08-22.

## Goal

Move ownership of the USB gadget out of `/etc/init.d/S03usbdev` and its four inlined imitations inside the Go server into one package, `server/service/presentation`, which compiles a declarative profile into an ordered list of ConfigFS operations and applies that list under a single lock. Program 1 is the foundation only: the profile schema, the pure compiler, the operation interface, the store, the migration, and the rewiring of the callers that shell out today. EDID and the Ethernet bridge get their own specs.

The bridge depends on this one. `/etc/init.d/S03usbdev stop` is `echo host > /proc/cviusb/otg_role` (`S03usbdev:162-163`), so it drops the entire gadget including the HID keyboard and mouse. Enslaving and releasing `usb0` into a bridge without losing the console requires something that can add and remove a single function symlink around a bind/unbind, and nothing in the tree can do that today.

Program 1 lands in two phases. Phase A reproduces today's gadget byte for byte and changes nothing observable on the wire, in `/dev`, or in any API response. Phase B is the behaviour fixes, in separate commits, each one named below.

## Why a clean transactional rewrite fails

The obvious shape for this is a manager that owns `g0`, tears it down, and rebuilds it from the profile on every apply. That shape breaks the keyboard.

`f_hid` allocates the `/dev/hidgN` minor from an `ida` at `mkdir functions/hid.GSn` time, and `hid/hid.go:29-32` hardcodes hidg0 as the keyboard, hidg1 as the relative mouse and hidg2 as the absolute pointer. `S03usbdev` never removes anything: `stop` unbinds the UDC and flips `otg_role`, and `start` re-runs over a live tree where `mkdir g0`, `mkdir strings/0x409`, `mkdir configs/c.1` and `mkdir functions/hid.GS*` all fail EEXIST, every `echo` overwrites, and every `ln -s` fails EEXIST. That is how `vm/virtual-device.go` adds RNDIS and mass storage without disturbing HID. Add-only idempotence is the contract the numbering rests on, so the transaction has to be expressed as bind, unbind, symlink add and symlink remove, never as `rmdir functions/*` or `rmdir g0`.

The second reason is that the shell script is still the boot-time configurator and will remain so for at least one release. `hid/status.go:24-27` implements HID-only mode by copying `ModeNormalScript` or `ModeHidOnlyScript` over `USBDevScript`, so the on-disk filename is never `S03usbhid`, and `support/sg2002/kvm_system/main/lib/system_init/system_init.cpp:99` unconditionally `cp -f`s the kvmapp copy back over `/etc/init.d/` on every app update, reverting HID-only mode. A manager that owns the gadget at runtime but leaves the boot path to a file-copy race has not fixed anything. Hence the write-through sentinel mirroring in D4 and the mode ownership in D3.

The third reason is that the current writes are fire and forget. Neither script has `set -e` and nothing checks `$?`, so several writes already fail invisibly on a shipping device. `echo e0 > functions/rndis.usb0/class` and its two siblings (`S03usbdev:66-68`) have no `0x` prefix, mainline `f_rndis.c` parses them with `kstrtou8(page, 0, ...)`, base 0 rejects unprefixed hex, and the values happen to equal the RNDIS IAD defaults so the failure leaves no trace. Correct Go that faithfully performs every write the script intends produces a different gadget from the one the script produces. This is why the acceptance criteria is an operation trace rather than a description of intent.

## Decisions

| # | Decision |
|---|---|
| D1 | No privileged helper process. The `Ops` interface is the privilege boundary. The server already runs as uid 0 and cannot drop privilege while it holds `/dev/hidg*`, GPIO and configfs, and two admin endpoints already grant a root shell, so a helper would guard a door in a wall with a hole in it. Argument in full below. |
| D2 | Phase A is a pure refactor. The compiled plan for each of the sixteen flag combinations emits a trace byte-identical to the shell script's. Every behaviour fix is Phase B, in its own commit. |
| D3 | Mode (normal against hid-only) becomes manager state instead of a filename. `SetHidMode` calls `presentation.SetMode`, which applies live where possible and stages plus reboots where not. |
| D4 | `/boot/usb.ncm`, `/boot/usb.rndis0` and `/boot/usb.disk0` are write-through mirrors for one release. Every apply rewrites them to match the active profile, so the script and the manager cannot disagree, and a downgrade or an OTA that restores an old `S03usbdev` still boots the user's gadget. |
| D5 | Migration preserves whatever the sentinels say, including the NCM-wins-over-RNDIS precedence at `S03usbdev:53,61`. NCM is never imposed on an RNDIS user. Choosing between them becomes an explicit profile choice in Program 3, on the Virtual Network control rather than on the bridge panel, since the choice decides what the gadget presents whether or not a bridge exists. The choice is those two and no more: ECM is not implemented in the gadget layer, so it is not offered. |
| D6 | `bcdDevice` stays unwritten in Phase A. `GetMode` works today off the kernel default; Phase B makes the marker explicit and reorders how mode is resolved. Detail below. |
| D7 | The capability table is `static-v0`, a plain data literal derived from what the shipping scripts demonstrably achieve, replaceable in a documented precedence order. It refuses plans, it never sizes them. |
| D8 | Built-in profiles are code. `standardProfile()` and `hidOnlyProfile()` return `Profile` values with byte-pinned report descriptors, written to disk for inspectability and always reconstructed from code, so a corrupt `standard.json` cannot brick USB and an OTA can fix a built-in. |
| D9 | The store lives at `/etc/kvm/presentation/`, mode 0600, using the `atomicFile` helper from `server/service/extensions/tunnel/config.go:78-152` verbatim. No sixth spelling of atomic write. |
| D10 | One gadget-level mutex, taken in exactly one helper, wrapping the existing HID quiesce bracket. The HID mutex is today the only mutual exclusion between four gadget mutators and must not be dropped while a new one is added. |
| D11 | The UI stays minimal. No fidelity badges, no taxonomy, no new modals. A profile either carries a captured descriptor tree or it does not, and that is a property the compiler reads. Modals are reserved for irreversible or self-locking actions, of which Program 1 has none. |
| D12 | Program 1 adds no routes. It rewires the handlers behind `/api/vm/device/virtual`, `/api/hid/mode`, `/api/hid/reset` and the storage mount path. |

## The privilege boundary

The plan this spec replaces called for a narrow privileged helper in front of configfs. Four grounds against building it now.

There is nothing to protect the helper from. `kvmapp/system/init.d/S95nanokvm`'s `start_services()` runs `/tmp/server/NanoKVM-Server &` directly from BusyBox `rcS` with no `su`, no `start-stop-daemon -c`, no capability bounding and no namespace. A helper reduces nothing unless the server drops privilege, and the server cannot drop privilege while it holds `/dev/hidg0-2` open for its process lifetime, writes `/sys/kernel/config/usb_gadget/**`, reads `/sys/class/gpio/gpio503-507/value` (`config/hardware.go:22-40`) and rewrites `/etc/init.d/S03usbdev` (`hid/status.go:182`).

The trust boundary a helper would create does not exist. `vm/terminal.go:44` starts `/bin/sh` under a pty bridged to a WebSocket. `vm/script.go:91-99` splices `req.Name` into `exec.Command("sh", "-c", ...)` with only `validate:"required"` on it, no suffix check on the run path unlike the upload path at `script.go:54-57`, no traversal check and no metacharacter check, so `x; id > /tmp/pwned` runs as root. Both are admin-gated at `router/vm.go:30-33`, and so are `SetHidMode` and `UpdateVirtualDevice`. Anyone who can reach the presentation API can already reach a root shell.

A second long-lived process breaks the lifecycle contract. `S95nanokvm`'s `stop_process` uses `killall -INT NanoKVM-Server` by process name and its ordering comment is load-bearing, since the server owns VI and VPSS and must release MMF before `kvm_system` stops. A helper daemon has to be added to that script, to `system_init.cpp`'s copy list and to the OTA install path, or `restart` orphans it.

The one component that could genuinely move is the store, and moving it breaks upgrades. Every `/etc/kvm/*` file the server writes is root-owned 0600, so a split that changes the writer's uid makes existing files unreadable on deployed devices with no migration path.

What makes the future split cheap, and is therefore built now:

- All configfs, exec and device access goes through `Ops`. No `os.WriteFile("/sys/kernel/config/...")` anywhere else in the package.
- `compile.go` is pure, imports nothing outside stdlib, and touches no filesystem.
- `Plan` is serializable. `[]Op` of `{Kind, Path, Target, Data}` marshals to JSON, so a future helper's protocol is already written down as "here is a validated Plan, execute it", and the helper never parses profiles or trusts user JSON.
- Paths in `Op` are relative to the gadget root and validated in `compile.go`: no `..`, no leading `/`, first segment in `{functions, configs, strings, os_desc, UDC}` or a known device attribute. That is the check a helper would need anyway.
- The `Ops` doc comment states the future IPC contract, including the four things that would not move: the pty, `/api/vm/script/run`, the `/dev/hidg*` file descriptors held for process lifetime, and GPIO.

Revisit when someone drops the server's privilege for real, which requires first killing or re-scoping `vm/terminal.go` and `vm/script.go`.

## What the gadget does today

### `/boot` sentinels read by `S03usbdev`

| Flag | Read at | Semantics | Written by Go today |
|---|---|---|---|
| `/boot/usb.vid` | `:14,16` | `cat` verbatim into `idVendor`; absent gives `0x3346` (L18) | no |
| `/boot/usb.pid` | `:20,22` | `cat` into `idProduct`; absent gives `0x1009` (L24) | no |
| `/boot/usb.ncm` | `:53` | selects `ncm.usb0`, absolute priority over rndis | no |
| `/boot/usb.rndis0` | `:61` | selects `rndis.usb0`, only in the `else` | yes, `vm/virtual-device.go:16,22-32` |
| `/boot/usb.disk0` | `:132,143` | dual purpose: existence enables mass_storage, contents select the backing file (empty gives `/dev/mmcblk0p3` L151, non-empty gives `cat` L153) | yes, `touch` and `rm` only, `virtual-device.go:17,33-45` |
| `/boot/usb.disk0.ro` | `:137` | writes `ro=1` (L139) and `cdrom=0` (L140) | no |
| `/boot/disable_hid` | `:84` | inverted gate on all three HID functions | no |
| `/boot/BIOS` | `:88,:103,:118` | writes `subclass=1` on all three HID functions | no |
| `/boot/usb.notwakeup` | `:92,:107,:122` | inverted gate on `wakeup_on_write=1` for all three | no |

Runtime state sits outside every flag file. `storage/image.go:22-25` writes `lun.0/{file,ro,cdrom,inquiry_string}` while the gadget is live, and those values partly survive a `stop`/`start` and are partly clobbered by it (H7). The manager models LUN state as runtime state distinct from profile state, or else a reapply silently produces a `cdrom=1` LUN advertising `USB Mass Storage` backed by `/dev/mmcblk0p3`.

### ConfigFS write inventory, standard profile

Ordered. Every path is relative to `/sys/kernel/config/usb_gadget/g0` after the `cd` at L7 and L10.

```
 1  mkdir g0 (EEXIST tolerated)                                  L9
 2  idVendor   <- /boot/usb.vid | "0x3346"                       L16/18
 3  idProduct  <- /boot/usb.pid | "0x1009"                       L22/24
    ** bcdUSB and bcdDevice NOT written **                        (H14)
 4  bDeviceClass=0xEF  bDeviceSubClass=0x02  bDeviceProtocol=0x01 L27-29
 5  mkdir strings/0x409                                          L31
 6  strings/0x409/serialnumber = "0123456789ABCDEF"              L32
 7  strings/0x409/manufacturer = "sipeed"                        L33
 8  strings/0x409/product      = "NanoKVM"                       L34
 9  mkdir configs/c.1                                            L36
10  configs/c.1/bmAttributes = 0xE0                              L37
11  configs/c.1/MaxPower     = 120                               L38
12  mkdir configs/c.1/strings/0x409                              L39
13  configs/c.1/strings/0x409/configuration = "NanoKVM"          L40
14  MAC derivation (globals, not local)                          L46-50
      usb_uid = head -c 4 of sha512sum(/sys/class/cvi-base/base_uid)
      dev  = 48:da:35:6e:<c1c2>:<c3c4>
      host = 48:da:35:6d:<c1c2>:<c3c4>
15  NCM branch (if /boot/usb.ncm):                               L55-59
      mkdir functions/ncm.usb0
      dev_addr, host_addr           (each guarded by [ -n ])
      os_desc/interface.ncm/compatible_id = "WINNCM"   (UNGUARDED)
      ln -s functions/ncm.usb0 configs/c.1/
    else RNDIS branch (if /boot/usb.rndis0):                     L63-71
      mkdir functions/rndis.usb0
      dev_addr, host_addr           (guarded)
      class=e0 subclass=01 protocol=03   ** no 0x prefix -> -EINVAL, no-op ** (H8)
      os_desc/interface.rndis/compatible_id     = "RNDIS"
      os_desc/interface.rndis/sub_compatible_id = "5162001"
      ln -s functions/rndis.usb0 configs/c.1/
16  MS-OS block if (ncm||rndis sentinel present, re-tested, H10)  L75-81
      os_desc/use = 1 ; os_desc/b_vendor_code = 0xCD
      os_desc/qw_sign = "MSFT100" ; ln -s configs/c.1 os_desc
17  HID, if ! /boot/disable_hid, in strict order GS0->GS1->GS2:   L84-130
      per function: mkdir -> [subclass=1 if /boot/BIOS]
                    -> [wakeup_on_write=1 unless /boot/usb.notwakeup]
                    -> protocol -> report_length -> report_desc -> ln -s  (ln LAST)
      GS0 protocol=1 len=8  desc=63B (LogMax/UsageMax 0xE7)
      GS1 protocol=2 len=4  desc=52B (3 buttons, rel X/Y/wheel)
      GS2 protocol=2 len=6  desc=74B (5 buttons, abs 0..0x7FFF X/Y, rel wheel)
18  mass_storage, if /boot/usb.disk0:                            L132-155
      mkdir functions/mass_storage.disk0
      ln -s ...  <-- LINK BEFORE ATTRS (legal only for mass_storage)  L135
      lun.0/removable = 1
      [if /boot/usb.disk0.ro] lun.0/ro=1 ; lun.0/cdrom=0
      lun.0/inquiry_string = "NanoKVM USB Mass Storage0520"  (exactly 28 B)
      lun.0/file = /dev/mmcblk0p3 | contents of /boot/usb.disk0
19  UDC <- `ls /sys/class/udc/ | cat`                            L157
20  /proc/cviusb/otg_role <- "device"                            L158
```

The final link order in `configs/c.1` is `{ncm|rndis}`, then `hid.GS0`, `hid.GS1`, `hid.GS2`, then `mass_storage.disk0`. That order fixes `bInterfaceNumber` assignment in the config descriptor and therefore host-side driver binding and interface naming, so it is reproduced exactly.

### ConfigFS write inventory, HID-only profile

`S03usbhid` deltas, everything else absent:

- `bcdUSB=0x0101` (L12) and `bcdDevice=0x0623` (L13). This is the only writer of `bcdDevice` in the tree.
- No `bDeviceClass`, `bDeviceSubClass` or `bDeviceProtocol`, no `serialnumber` (L29 is commented out), no MAC derivation, no network function, no `os_desc`, no mass storage, no `disable_hid` gate.
- `configs/c.1/bmAttributes=0xA0` and `MaxPower=200`.
- HID `subclass=1` is written unconditionally on all three functions, with no `/boot/BIOS` test.
- The GS0 report descriptor differs: bytes at offsets 52 and 58 are `25 65` and `29 65` (Logical and Usage Maximum 101) against `25 E7` and `29 E7` (231), so HID-only mode physically cannot report usages 0x66 through 0xE7. The GS2 descriptor differs too: three buttons (`29 03`, `95 03`, padding `75 05`) against five.

These are two functionally different HID capabilities and are modelled as two distinct compiled report descriptors, pinned in Go as `[]byte` literals with a test asserting the exact bytes and the lengths 63, 52 and 74 for each set. `S03usbdev`'s GS1 `report_desc` uses single-nibble escapes (`\x5\x1\x9\x2...`) and BusyBox `echo -ne` consumes up to two hex digits per escape, so transcribe the decoded bytes rather than the escape text.

## Ordering and kernel contracts

1. **CWD is part of the call.** configfs `get_target()` resolves symlink targets with `kern_path()` against the calling process's CWD, so `ln -s functions/hid.GS0 configs/c.1` works only because L7 and L10 did `cd`. A relative `os.Symlink` from elsewhere returns ENOENT. Either `Chdir` into `g0` or pass absolute targets.
2. **`ln -s` last** for `hid.*`, `ncm.*` and `rndis.*`. All f_hid, f_ncm and f_rndis option stores return `-EBUSY` once `opts->refcnt > 0`, meaning once the function is linked. `report_desc`, `report_length`, `protocol`, `subclass`, `dev_addr`, `host_addr` and `class` are writable only before the link.
3. **mass_storage is the exception.** The LUN attributes have no refcnt check, so the link may precede them. `lun.0/ro` and `lun.0/cdrom` still return `-EBUSY` while `lun.0/file` is open. The order is close the file (write `"\n"`), then `ro`, then `cdrom`, then `file`. `image.go:73-93` already encodes this.
4. **`mkdir` order GS0, GS1, GS2** allocates the `/dev/hidg0`, `hidg1`, `hidg2` minors from an `ida` at mkdir time, and `hid/hid.go:29-32` hardcodes the mapping. Never reorder, never tear down and rebuild.
5. **Add-only idempotence is the contract.** `stop` unbinds only, `start` re-runs over a live tree, and the transaction is bind and unbind plus symlink add and remove.
6. **Bind preconditions.** Linking a function into a bound config returns `-EBUSY`, and writing a non-empty UDC while bound returns `-EBUSY`. Every mutation is bracketed by an unbind.
7. **UDC name.** Read `/sys/class/udc/`, require exactly one entry, and fail loudly otherwise. A zero-byte write is an unbind, and that is the current silent total-failure mode at L157, at L169 and after `restart_phy` (H4).
8. **String store semantics.** `usb_string_copy()` strips one trailing newline, `fsg_store_inquiry_string()` truncates at the first newline, and numeric stores use `kstrtoXX(page, 0, ...)`, so a `0x` prefix is mandatory for hex.
9. **`inquiry_string` is exactly 28 bytes**, being `%-8s%-16s%04x`. Twenty-nine returns `-EINVAL`.
10. **`os_desc/use` is never cleared today** (H11) and its trigger is decoupled from function creation (H10). The compiled profile derives `os_desc` from what functions are actually in the plan and clears it when no MS-OS function is present. This is a deliberate behaviour change and lands in Phase B rather than during the pure refactor.

## Hazards

These are the reasons the shell scripts look the way they do. Each one is a comment in the Go that replaces it.

| # | Hazard |
|---|---|
| H1 | CWD is program state. The `cd` at `S03usbdev:7,10` is unchecked, so if `S01fs:17` failed to mount configfs the script keeps running in init's CWD, `mkdir g0` creates a real directory there, and every subsequent write scatters files across the rootfs. Nothing detects it. |
| H2 | `ln -s` must come last per HID function. Reordering enumerates three 64-byte HID devices carrying the kernel's default report descriptor, with no error anywhere. |
| H3 | The `mkdir` EEXIST failures are load-bearing. `stop` removes nothing, so every `start` after the first re-runs over a live tree, and that is how `vm/virtual-device.go` adds RNDIS and mass storage without renumbering `/dev/hidgN`. |
| H4 | `ls /sys/class/udc/ \| cat > UDC` writes zero bytes when the directory is empty, and configfs treats that as unbind. A fully configured gadget silently never binds, at `S03usbdev:157`, `:169` and after `restart_phy`. |
| H5 | `restart_phy` (`S03usbdev:188-190`) unbinds the dwc2 platform driver, sleeps a fixed second, and rebinds. There is no readiness poll and no retry, so death or a slow re-probe inside that window leaves USB dead until reboot. |
| H6 | `stop`, or a `stop_start` interrupted after the stop, leaves the machine with `otg_role=host` and an unbound gadget. The attached PC loses keyboard and mouse and no watchdog brings it back. |
| H7 | Runtime LUN state partly survives a `stop`/`start`. L136 and L142 rewrite `removable` and `inquiry_string`, destroying the CD-ROM inquiry string `image.go:103` set, and L151/L153 rewrite `file`, while `ro` and `cdrom` are left as `image.go` set them unless `/boot/usb.disk0.ro` exists. |
| H8 | `echo e0/01/03 > functions/rndis.usb0/{class,subclass,protocol}` (`S03usbdev:66-68`) lack the `0x` prefix, `kstrtou8(page, 0, ...)` rejects unprefixed hex, and the values coincide with the RNDIS IAD defaults, so the failure is invisible. Read the live attribute values off a running device before deciding what Go should write. |
| H9 | Every write is fire and forget. Nothing checks `$?`, a partially configured gadget still gets bound and enumerated, and `restart_usb_dev` prints `USB Restart OK!` unconditionally even when both UDC writes failed. |
| H10 | The MS-OS block's trigger at `S03usbdev:75` re-tests the sentinels rather than tracking whether a network function was created, so `os_desc/use=1` and the `os_desc/c.1` symlink can be applied to a gadget with no network function at all. |
| H11 | `os_desc/use` is never cleared. `virtual-device.go:28-31` removes the rndis symlink and the sentinel but leaves `use=1` and the `os_desc/c.1` link, so the gadget keeps answering the 0xEE string request forever. |
| H12 | `usb_dev_mac` and `usb_host_mac` are globals rather than `local` (`S03usbdev:46-50`). Harmless with one call per invocation, and a trap for anyone who adds a second `start_usb_dev` call to a case arm. |
| H13 | The gadget MAC carries 16 bits of entropy, from the first four hex characters of `sha512sum` of `/sys/class/cvi-base/base_uid`, and the serial number is the fleet-wide constant `0123456789ABCDEF`. Windows keys device-instance state on VID, PID and serial, so two NanoKVMs on one host collide. If `base_uid` is missing the `[ -n ]` guards skip both addresses and the kernel assigns a random MAC on every bind, the regression the `S03usbdev:42-45` comment exists to prevent. |
| H14 | `S03usbdev` has never written `bcdDevice` in any revision, and `hid/status.go:22,31` uses the kernel default `0x0510` as the normal-mode sentinel. A rewrite that writes any `bcdDevice` makes `GetMode()` return `invalid mode flag` and breaks the HID mode UI. |
| H15 | `report_length` and the descriptor must agree. GS0 8, GS1 4, GS2 6 are derived from the descriptors (8 = 1+1+6, 4 = 1+1+1+1, 6 = 1+2+2+1), `report_length` also sets the interrupt-IN `wMaxPacketSize`, and `hid.go` writes exactly that many bytes per report. |

## Package layout

House style is `server/service/extensions/tunnel/config.go`, the newest config code in the tree. Follow it rather than the four inlined copies.

```
server/service/presentation/
  profile.go        # schema + validation + the two built-in profiles
  profile_test.go
  store.go          # atomic store over /etc/kvm/presentation/, tunnel-config.go shape
  store_test.go
  capability.go     # CapabilityTable, snapshot, static v0 table + probe hook
  endpoint.go       # allocator: budget accounting over compiled functions
  compile.go        # Profile -> Plan (ordered []Op). Pure. No I/O.
  compile_test.go   # golden traces
  ops.go            # type Ops interface, the privilege boundary
  ops_configfs.go   # in-process implementation (root, direct syscalls)
  ops_record.go     # recording/fake Ops for tests
  apply.go          # transaction: snapshot -> quiesce -> unbind -> mutate -> bind -> verify -> LKG
  snapshot.go       # live configfs read-back
  migrate.go        # /boot/usb.* -> profile, one-shot, idempotent
  manager.go        # singleton, sync.Once, mutexes; the public API
  service.go        # gin handlers (Program 1 adds none; wires existing ones)
```

## Key types

```go
// profile.go: the schema, versioned, JSON, stored at /etc/kvm/presentation/<name>.json
type Profile struct {
    SchemaVersion int      `json:"schema_version"` // 1
    Name          string   `json:"name"`           // "standard" | "hid-only" | user
    BuiltIn       bool     `json:"built_in"`
    Device        Device   `json:"device"`
    Config        ConfigDesc `json:"config"`
    Functions     []Function `json:"functions"`    // ORDER IS THE LINK ORDER
    OSDesc        *OSDesc  `json:"os_desc,omitempty"`
}

type Device struct {
    VendorID   string  `json:"vendor_id"`            // "0x3346"
    ProductID  string  `json:"product_id"`           // "0x1009"
    BCDUSB     *string `json:"bcd_usb,omitempty"`    // nil = DO NOT WRITE (normal mode)
    BCDDevice  *string `json:"bcd_device,omitempty"` // nil = DO NOT WRITE
    Class      *uint8  `json:"class,omitempty"`      // 0xEF; nil for hid-only
    SubClass   *uint8  `json:"subclass,omitempty"`
    Protocol   *uint8  `json:"protocol,omitempty"`
    Serial     *string `json:"serial,omitempty"`     // nil for hid-only (L29 commented out)
    Manufacturer string `json:"manufacturer"`
    Product      string `json:"product"`
}
```

The pointer-means-do-not-write distinction is load-bearing. `bcdDevice`'s absence is today's normal-mode sentinel (H14) and HID-only genuinely omits `serialnumber`, so a non-pointer field with a zero value would write `0x0000` and break `GetMode`.

```go
type Function struct {
    Kind     FunctionKind `json:"kind"`      // hid | ncm | rndis | mass_storage
    Instance string       `json:"instance"`  // "GS0","GS1","GS2","usb0","disk0"
    HID      *HIDFunction `json:"hid,omitempty"`
    Net      *NetFunction `json:"net,omitempty"`
    Storage  *StorageFunction `json:"storage,omitempty"`
}

type HIDFunction struct {
    Protocol      uint8  `json:"protocol"`
    SubClass      uint8  `json:"subclass"`      // 1 = boot
    ReportLength  uint16 `json:"report_length"`
    WakeupOnWrite bool   `json:"wakeup_on_write"`
    ReportDesc    []byte `json:"report_desc"`   // base64; validated len == expected
    DevNodeIndex  int    `json:"-"`             // derived: creation ordinal -> /dev/hidgN
}
```

`Validate()` enforces H15: `ReportLength` equals the length computed from `ReportDesc`.

```go
// ops.go: the ONLY place syscalls/exec happen. This is the split boundary.
type Ops interface {
    Mkdir(rel string) error                 // EEXIST is success
    WriteFile(rel string, data []byte) error
    ReadFile(rel string) ([]byte, error)
    Symlink(target, linkRel string) error   // EEXIST is success
    Remove(rel string) error                // unlink a config symlink only
    ListUDC() ([]string, error)
    BindUDC(name string) error
    UnbindUDC() error
    SetOTGRole(role string) error           // /proc/cviusb/otg_role
    ResetPHY(ctx context.Context) error     // dwc2 unbind/poll/bind
}
```

Every method takes a path relative to `g0`. The configfs implementation holds an `*os.File` dirfd on `g0` and uses `openat`, `symlinkat` and `unlinkat`, which sidesteps H1 and constraint 1 without a process-global `Chdir`. `os.Symlink` has no `-at` variant in stdlib, so use `unix.Symlinkat(target, int(dirfd.Fd()), link)`.

```go
// compile.go: pure
type OpKind int // OpMkdir, OpWrite, OpSymlink, OpUnlink, OpBind, OpUnbind, OpOTGRole
type Op struct { Kind OpKind; Path string; Target string; Data []byte }
type Plan struct { Ops []Op; Endpoints EndpointUse; Profile string }
func Compile(p Profile, caps CapabilityTable) (Plan, error)
```

```go
// manager.go
type Manager struct {
    store   *Store
    ops     Ops
    caps    CapabilityTable
    mu      sync.Mutex   // gadget-level lock; replaces the incidental HID mutex role
    active  string       // active profile name
}
func GetManager() *Manager           // sync.Once singleton, like controlmode.GetManager()
func (m *Manager) Snapshot() (Snapshot, error)
func (m *Manager) Apply(ctx context.Context, name string) error
func (m *Manager) Rebind(ctx context.Context) error
func (m *Manager) ResetPHY(ctx context.Context) error
```

## Capability table

Phase 0 has not run, so there is no measured endpoint budget. The table is named `static-v0` and derived from what the shipping scripts demonstrably achieve. The maximal simultaneous configuration `S03usbdev` builds is `{ncm|rndis}` plus `hid.GS0`, `hid.GS1`, `hid.GS2` plus `mass_storage.disk0`. Counting kernel 5.10 bind behaviour, `f_ncm` takes one bulk IN, one bulk OUT and one interrupt IN, `f_rndis` the same, `f_hid` allocates both an interrupt IN and an interrupt OUT unconditionally in `hidg_bind()`, and `f_mass_storage` takes one bulk IN and one bulk OUT. That totals six IN, five OUT, plus EP0.

The device tree agrees with the six. `build/boards/default/dts/sg200x/soph_base.dtsi` in the LicheeRV-Nano-Build tree declares `g-rx-fifo-size = <536>`, `g-np-tx-fifo-size = <32>` and `g-tx-fifo-size = <768 512 512 384 128 128>`, so the dwc2 controller has six dedicated IN FIFOs plus EP0, and today's maximal function set already consumes all six. The real `num_dev_ep` is readable only from the hardware `GHWCFG` registers, not from `/sys/class/udc/*`, so the budget cannot be probed at runtime. Availability can be: `mkdir functions/ncm.probe` in a scratch gadget directory at `/sys/kernel/config/usb_gadget/g_probe` that is never bound succeeds only if `usb_f_ncm` is loadable. Never probe `hid.*` that way, because the probe consumes a `/dev/hidgN` minor and shifts the numbering `hid/hid.go:29-32` depends on.

```go
// capability.go
type CapabilityTable struct {
    Source      string   `json:"source"`       // "static-v0" | "probe-v1" | "measured"
    GeneratedAt time.Time `json:"generated_at"`
    MaxInEndpoints  int   `json:"max_in_endpoints"`   // 6
    MaxOutEndpoints int   `json:"max_out_endpoints"`  // 5
    Functions   map[FunctionKind]FunctionCaps `json:"functions"`
}
type FunctionCaps struct {
    Available bool `json:"available"` // static-v0: true for hid/ncm/rndis/mass_storage
    InEPs, OutEPs int
    Attributes map[string]bool // e.g. "os_desc/interface.ncm" present
}

// Replaceable, in this precedence order:
//   1. /etc/kvm/presentation/capability.json  (written by a future Phase-0 probe)
//   2. probeAvailability() results merged onto staticV0 (availability only, never budget)
//   3. staticV0
func LoadCapabilities() CapabilityTable
```

The allocator does exactly one thing in Program 1: refuse to compile a plan that exceeds the demonstrated budget, and name the function it refused for. It is permissive about the two built-ins by construction, since `standard` at full flag expansion is defined to be the budget, so neither built-in can be refused by its own table. Every guardrail is inert during the refactor and bites only when Program 3 adds a second network function or a user adds a fourth HID. `Source` is carried into every error message and into the snapshot API, so the message reads `rejected by capability table static-v0`. `staticV0` is a plain data literal in one file, with `capability_test.go` asserting that `Compile(standardProfile(), staticV0)` and `Compile(hidOnlyProfile(), staticV0)` both succeed with zero headroom warnings.

## Store

`grep -rn "presentation" server/` returns nothing, so the directory name is free.

```
/etc/kvm/presentation/
  active                 # single line: profile name                       0600
  standard.json          # built-in, rewritten on every boot from code     0600
  hid-only.json          # built-in                                        0600
  <user>.json            # user profiles                                   0600
  .last-known-good       # name of the last profile that bound + verified  0600
```

The directory is 0755 and every file is 0600. The shape is copied from `tunnel/config.go:14-42`: a package-level `var presentationDir = "/etc/kvm/presentation"`, declared `var` rather than `const` so `useTestConfigDir(t)` can swap it the way `tunnel/config_test.go:13-19` does, plus a `var configMu sync.Mutex`, with an exported `PresentationDir` constant kept for other packages. Writes go through the `atomicFile` helper at `tunnel/config.go:78-152` verbatim, which chmods the temporary file, renames, re-chmods the destination, propagates the `Sync` error and swallows only the directory `Open` error.

Two tests exist because their absence reads as an incomplete port: `TestSaveProfilePermissions` asserting `info.Mode().Perm() == 0o600`, mirroring `mcp/config_test.go:42-49`, and `TestLoadProfileRejectsCorruptJSON` writing `[]byte("{")`, mirroring `mcp/config_test.go:62-71`. A missing file yields the zero value or the built-in default with a nil error; corrupt JSON yields a wrapped error.

Built-in profiles are code rather than data. `standardProfile()` and `hidOnlyProfile()` return `Profile` values carrying the byte-pinned report descriptors, and they are written to disk on boot for inspectability but always reconstructed from code on load, so an OTA can fix them and a corrupt `standard.json` can never brick USB.

## Callers that move behind the manager

| Caller | Today | Post-refactor |
|---|---|---|
| `vm/virtual-device.go:20-45,106` | four literal `sh -c` lists invoking `S03usbdev stop/start` plus `rm -rf configs/c.1/<fn>` | `presentation.Apply(ctx, profileWith(net/disk toggled))` |
| `vm/virtual-device.go:54-57` | state is `os.Stat` on two `/boot` files | state is the manager's `Snapshot()`, which reports whether a function is actually linked, fixing the NCM-shadows-RNDIS lie |
| `hid/status.go:67-108` `SetHidMode` | copy script plus `reboot` | `presentation.SetMode(normal\|hid-only)`, applied live where possible, staged plus reboot where not |
| `hid/status.go:146-162` `ResetUSBPHY` | `sh -c "S03usbdev restart_phy"` | `manager.ResetPHY(ctx)` with a real readiness poll in place of `sleep 1` (H5) |
| `hid/status.go:226-241` `GetMode` | reads `bcdDevice` and string-maps it | reads the manager's active profile, with `bcdDevice` demoted to a written marker |
| `storage/image.go:123-144` | a fourth independent gadget restart, raw UDC unbind and rebind with two 100 ms sleeps | `manager.Rebind(ctx)` |
| `storage/image.go:57-148` LUN writes | direct sysfs writes with no check that the disk gadget exists | keep the writes, gate them on `snapshot.HasFunction("mass_storage.disk0")` so the UI stops seeing an opaque `-2` |
| the `hid.Lock()`/`CloseNoLock()`/`OpenNoLock()` bracket at `status.go:85-91`, `image.go:123-129` and `virtual-device.go:97-103` | copy-pasted three ways, two of which discard the reopen error | one `manager.withHIDQuiesced(fn)` helper using `OpenNoLockWithRetry(2s, 100ms)` everywhere, since `virtual-device.go` currently uses the no-retry variant |

The HID mutex is today the only mutual exclusion between `UpdateVirtualDevice` and `MountImage`. There is no gadget-level lock at all. The manager introduces an explicit one and does not drop the incidental HID one.

## API and compatibility

Program 1 adds no routes. The wire contract of `/api/vm/device/virtual` is frozen.

`GET` keeps returning `{"code":0,"msg":"success","data":{"network":bool,"media":false,"disk":bool}}`. The `media` field stays, even though `proto/vm.go:69` never assigns it and nothing reads it, because removing it is a visible JSON shape change for unknown third-party clients and buys nothing. `POST` keeps binding `UpdateVirtualDeviceReq{Device string}` with no `json` tag, working only through `encoding/json`'s case-insensitive match at `proto/vm.go:73-75`, so the field is not renamed and the binder is not changed. The response stays `{"on": <fresh re-check>}` and the `-1`, `-2` and `-3` error codes keep their exact strings.

`UpdateVirtualDevice` becomes a thin adapter: load the active profile, add or remove the `rndis.usb0` entry (or `ncm.usb0`, preserving whichever is currently active) and the `mass_storage.disk0` entry, then call `manager.Apply`.

Three behaviour changes ride along, all invisible to the current frontend, which only `console.log`s failures at `virtual-devices.tsx:49-66`. `network` becomes true only when a network function is actually linked into `configs/c.1`, which fixes the case where `/boot/usb.ncm` exists, `S03usbdev:53` takes the NCM branch, `rndis.usb0` is never linked and `GetVirtualDevice` reports `network: true` anyway. The backend enforces the HID-only precondition that only `virtual-devices.tsx:68-79` checks today, since a direct API call currently creates flag files that `S03usbhid` ignores entirely. And the storage endpoints gate on a real snapshot rather than writing into a LUN that may not exist.

## The `/dev/hidg*` numbering invariant

`functions/hid.GS0`, `hid.GS1` and `hid.GS2` are created by `mkdir` in that order, once, and never removed. The minor comes from an `ida` at mkdir time and `hid/hid.go:29-32` hardcodes hidg0 as keyboard, hidg1 as relative mouse and hidg2 as absolute pointer.

Rules enforced in code:

- `Apply` never calls `rmdir` on `functions/*` and never removes `g0`. Path validation restricts `Ops.Remove` to symlinks under `configs/c.1/`.
- `Compile` emits the HID `mkdir` ops in profile-array order, and `Profile.Validate()` rejects any profile whose HID functions are not exactly `GS0`, `GS1`, `GS2` in that order with `DevNodeIndex` 0, 1, 2.
- Nothing probes `hid.*` in the scratch gadget, because the probe consumes a minor.
- A test asserts that applying standard, then hid-only, then standard emits zero `rmdir` or `unlink` ops touching `functions/hid.*`.

Post-apply verification reads back the presence of `/dev/hidg0` through `hidg2` using `OpenNoLockWithRetry(2s, 100ms)`, the variant `hid/status.go:157` uses, rather than the bare `OpenNoLock()` at `virtual-device.go:101` whose error is discarded.

## Migration

Migration reads the current on-disk truth and preserves it:

```
if /boot/usb.ncm exists            -> profile net function = ncm.usb0
else if /boot/usb.rndis0 exists    -> profile net function = rndis.usb0
else                               -> no net function
```

which matches `S03usbdev:53,61` exactly, NCM-wins precedence included. It runs once and is idempotent: it fires only when `/etc/kvm/presentation/active` is absent, writes the derived profile, sets `active`, and leaves the `/boot` sentinels where they are.

The sentinels stay for one release because `system_init.cpp:99` may restore an old `S03usbdev` over `/etc/init.d/` on an OTA, and a downgrade must still boot with the user's network gadget. For that release the manager treats them as write-through mirrors, rewriting `/boot/usb.ncm`, `/boot/usb.rndis0` and `/boot/usb.disk0` on every apply to match the active profile, so the script and the manager cannot disagree about what the gadget should be. Removing the sentinels is a separate later change, gated on the init script no longer being the boot-time configurator.

A migration fixture per starting state, covering `{none, ncm, rndis, ncm+rndis, disk, disk+rndis}`, asserts that the derived profile compiles to a trace byte-identical to the pre-migration script trace for that same flag set.

## `bcdDevice`

`S03usbdev` has never written `bcdDevice` in any revision, and `GetMode` works today anyway. The gadget core's `get_default_bcdDevice()` computes `bin2bcd(VERSION) << 8 | bin2bcd(PATCHLEVEL)`, yielding `0x0510` on Linux 5.10, and `status.go:31` maps that string to normal mode. The hazard is that the normal-mode marker is an implicit kernel-version artifact, so a vendor bump to 5.15 silently yields `0x050f` or `0x0515`, `GetMode` returns `invalid mode flag`, and three things break at once: `/api/hid/mode`, `/api/storage/image/mounted` (`image.go:153-157` hard-fails the endpoint on a `GetMode` error), and `SetHidMode`'s already-in-that-mode short circuit at `status.go:80`, which drops the error, so `mode` is `""`, the short circuit never matches, and the handler copies the script and reboots.

Phase A sets `standardProfile().Device.BCDDevice = nil`, the manager writes nothing, the golden trace asserts zero `bcdDevice` writes, and `GetMode` is untouched.

Phase B, as its own commit, sets `BCDDevice = ptr("0x0510")` so the marker becomes explicit, and resolves mode in three tiers: the manager's active profile first, `bcdDevice` exact match second, `0x05xx`-tolerant parsing third. The same commit surfaces the `GetMode` error at `status.go:80` instead of dropping it. `S03usbhid`'s `0x0623` is preserved verbatim as the HID-only marker.

## Files

| Path | Action |
|---|---|
| `server/service/presentation/*.go` | new, the fourteen files listed above |
| `server/service/vm/virtual-device.go` | modify: four `sh -c` lists and the `os.Stat` state check become manager calls |
| `server/service/hid/status.go` | modify: `SetHidMode`, `ResetUSBPHY`, `GetMode` move behind the manager |
| `server/service/storage/image.go` | modify: gadget restart becomes `Rebind`, LUN writes gated on a snapshot |
| `server/service/hid/hid.go` | unchanged, and the reason for the numbering invariant |
| `kvmapp/system/init.d/S03usbdev` | unchanged in Program 1, still the boot-time configurator |
| `kvmapp/system/init.d/S03usbhid` | unchanged in Program 1, still the source of the HID-only descriptors |
| `web/src/pages/desktop/menu/settings/*` | unchanged in Program 1 |

## Risks

**R1.1 A transactional apply tears down `functions/hid.*` and renumbers `/dev/hidgN`.** The server then writes mouse reports to the keyboard endpoint, silently, visible only as garbage input on the attached host. Retires when the golden-trace suite covers standard, hid-only, standard, standard-plus-disk in sequence and asserts zero ops matching `rmdir functions/*` or `unlink functions/hid.*`, that every `unlink configs/c.1/hid.*` an apply emits is linked again before the transaction ends, and when an on-device run captures `ls -l /dev/hidg*` minors before and after ten apply cycles and finds them identical. The config symlink is not part of this risk: `hidg_alloc_inst` takes the minor from the `ida` at `mkdir` and `hidg_free_inst` returns it at `rmdir`, while the `opts->refcnt` that makes an attribute store `-EBUSY` is moved by `hidg_alloc` and `hidg_free` on link and unlink, so releasing it costs no minor. `TestKernelTier2UnlinkKeepsHIDMinors` asserts that against the kernel.

**R1.2 The pure refactor is not actually byte-identical, and the divergence is a write the kernel silently rejects.** The scripts have no `set -e` and several writes already fail invisibly (H8's unprefixed `class=e0`), `os_desc/use=1` is set on gadgets with no network function (H10) and never cleared (H11), so today's real behaviour includes those bugs and correct-looking Go can produce a different gadget. Retires when the golden traces match, and when an on-device diff is empty for all sixteen flag combinations across three captures taken before and after the swap: `find /sys/kernel/config/usb_gadget/g0 -type f -exec sh -c 'echo "$1: $(cat "$1" 2>/dev/null)"' _ {} \;`, `ls -l configs/c.1/`, and `lsusb -v` from an attached host. The `lsusb -v` diff is the one that catches a faithfully reproduced write that was always a no-op.

**R1.3 Losing the incidental serialization the HID mutex provides today.** `h.Lock()` is currently the only mutual exclusion between `UpdateVirtualDevice` (`virtual-device.go:97-103`), `MountImage` (`image.go:123-129`), `SetHidMode` (`status.go:85-91`) and `ResetUSBPHY` (`status.go:147-161`), so adding a gadget-level lock while relaxing the HID bracket, or taking the two in different orders on different paths, gives either a deadlock or a concurrent `-EBUSY` storm. Retires when a single `manager.withGadgetLock(func() { withHIDQuiesced(...) })` helper is the only place either lock is taken, `go vet` and `go test -race` pass a stress test hammering all four entry points concurrently, and a grep shows zero remaining call sites of `hid.Lock()` outside `presentation/` and `ws/client.go`.

## Testing

The acceptance criteria for Phase A is an operation trace, defined as an ordered list of `{op, path, value}` tuples. `Compile` is pure and emits one directly, so the whole compiler runs on an x86 dev machine with no device and no configfs.

Golden traces are generated from the shell scripts rather than written by hand. Run `S03usbdev start` and `S03usbhid start` under a PATH shim in which `mkdir`, `ln`, `cat`, `echo` and `ls` are replaced by stubs that append their arguments to a trace file, against a fake `/sys/kernel/config` tree and a fake `/boot` populated for each flag combination. Whatever the script actually does, including the writes that would have failed on a real kernel, ends up in the trace, so the Go compiler is compared against observed behaviour rather than against a reading of the script.

The matrix is sixteen traces: `{normal, hid-only}` by `{no-net, ncm, rndis, ncm+rndis}` by `{disk off, disk on}`, plus single-variable deltas for `/boot/BIOS`, `/boot/usb.notwakeup`, `/boot/disable_hid` and `/boot/usb.disk0.ro`. Each trace asserts the exact op order, the exact bytes written including whether a trailing newline is present, the symlink target strings, and that `bcdDevice` and `bcdUSB` appear zero times in the normal-mode trace until the Phase B change lands.

Around that: `profile_test.go` asserts the byte-pinned report descriptors and their lengths, 63, 52 and 74 for each of the two sets, and asserts H15's agreement between `ReportLength` and the descriptor. `store_test.go` carries the 0600 permission test and the corrupt-JSON test. `capability_test.go` asserts both built-ins compile against `staticV0` with zero headroom. `migrate_test.go` runs the six starting states. The `ops_record.go` fake is the only `Ops` implementation any test uses, so no test touches a real `/sys`.

No CI workflow in this repo runs `go test` or `go vet` today, so these run locally and in review until that changes.

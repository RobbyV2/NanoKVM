# FunctionFS Hybrid

Hybrid keeps `hid.GS0` and `hid.GS1`, removes optional links, and adds only `ffs.hybrid`. Never remove persistent HID function directories.

Treat imported descriptors, strings, reports, transfer lengths, statuses, and timing as hostile. Parse and account the full layout before mounting FunctionFS or changing ConfigFS. Keep descriptor, control, transfer, interface, endpoint, and aggregate in-flight limits load-bearing in tests.

Supported transfers are control, bulk, and interrupt at alternate setting zero. Isochronous, hubs, device-level classes, nonempty BOS capabilities, ambiguous CDC/IAD ownership, fixed endpoints, and layouts outside the measured DWC2 endpoint/FIFO table are Exact-only.

USBFS retains URB and buffer pointers after `ioctl`. Pin both until reap or a synchronizing file close. Never complete or unpin a request merely because discard failed.

Cleanup order is unbind, unlink `configs/c.1/ffs.hybrid`, restore the persistent presentation, close and unmount FunctionFS, detach VHCI, then clear the recovery marker. Hardware proof still gates enumeration, resets, stalls, and sustained throughput.

The descriptor block sets `FUNCTIONFS_VIRTUAL_ADDR` (flags bit 4), so the kernel names the endpoint files `ep<addr>` after the addresses the compiler assigned — `ep01`, `ep81` — rather than `ep1`, `ep2` by index. Open them by `functionFSEndpointName(address)`, never by enumeration order, and open them while `ep0` is still held: closing `ep0` destroys the endpoint files.

`kernel_tier2_test.go` holds the ordering contract to a real kernel: `mount -t functionfs hybrid` is `ENODEV` until `functions/ffs.hybrid` exists in configfs, `ENOENT` under a name no instance carries, and `functionfs` is absent from `/proc/filesystems` entirely until the first instance, so a preflight that greps it reads the wrong thing. See `service/presentation/CLAUDE.md` for how the tier runs.

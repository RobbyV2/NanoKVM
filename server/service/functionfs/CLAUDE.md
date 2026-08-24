# FunctionFS Hybrid

Hybrid keeps `hid.GS0` and `hid.GS1`, removes optional links, and adds only `ffs.hybrid`. Never remove persistent HID function directories.

Treat imported descriptors, strings, reports, transfer lengths, statuses, and timing as hostile. Parse and account the full layout before mounting FunctionFS or changing ConfigFS. Keep descriptor, control, transfer, interface, endpoint, and aggregate in-flight limits load-bearing in tests.

Supported transfers are control, bulk, interrupt, and isochronous. Hubs, audio, device-level classes, nonempty BOS capabilities, ambiguous CDC/IAD ownership, fixed endpoints, and layouts outside the measured DWC2 endpoint/FIFO table are Exact-only.

An interface with alternate settings is presented as exactly two: its zero-bandwidth alternate 0 and the one streaming alternate whose `wMaxPacketSize * mult` is widest inside the controller's deepest dedicated IN FIFO. That is not a simplification, it is forced twice over. `f_fs` names an endpoint file after the address in the descriptor, so a second alternate carrying an endpoint at the same address collides on the same name, and `MAX_ALT_SETTINGS` is 2. `selectAlternates` refuses an interface whose alternate 0 is not zero-bandwidth rather than picking one anyway.

`set_alt` enables every endpoint at once and hands userspace no alternate number, so stream start is **never** inferred from an event. UVC sends `SET_CUR` on `VS_COMMIT_CONTROL` immediately before `SET_INTERFACE`, `FUNCTIONFS_ALL_CTRL_RECIP` forwards it, and its `dwMaxPayloadTransferSize` sizes the slot. `Relay.videoCommit` is the only trigger.

Isochronous endpoints do not go through `transferLoop`. `isochronousStream` keeps `isoTransfersInFlight` usbfs URBs of `isoPacketsPerTransfer` microframes each in flight on the source and submits their packets to the FunctionFS endpoint through `io_submit`, one iocb per microframe. The mmap'd, mlocked pool is simultaneously the usbfs transfer buffer and the aio buffer, so a payload is never copied. Neither loop frees anything: `stop` is the only release, and only after both loops have exited and the kernel has returned every URB and every iocb. A source transfer that outlives its cancellation leaks the whole stream — `ErrAIOLeaked` — rather than letting `munmap` race a DMA.

Both loops block in a syscall, which is what makes `raiseRealtime` safe. `sched_rt_runtime_exceeded` sits outside `CONFIG_RT_GROUP_SCHED`, so SCHED_FIFO on a thread that spins costs 50 ms of every second, which is 400 lost microframes. Do not put a polling loop under it.

USBFS retains URB and buffer pointers after `ioctl`. Pin both until reap or a synchronizing file close. Never complete or unpin a request merely because discard failed.

Cleanup order is unbind, unlink `configs/c.1/ffs.hybrid`, restore the persistent presentation, close and unmount FunctionFS, detach VHCI, then clear the recovery marker. Hardware proof still gates enumeration, resets, stalls, and sustained throughput.

The descriptor block sets `FUNCTIONFS_VIRTUAL_ADDR` (flags bit 4), so the kernel names the endpoint files `ep<addr>` after the addresses the compiler assigned — `ep01`, `ep81` — rather than `ep1`, `ep2` by index. Open them by `functionFSEndpointName(address)`, never by enumeration order, and open them while `ep0` is still held: closing `ep0` destroys the endpoint files.

`kernel_tier2_test.go` holds the ordering contract to a real kernel: `mount -t functionfs hybrid` is `ENODEV` until `functions/ffs.hybrid` exists in configfs, `ENOENT` under a name no instance carries, and `functionfs` is absent from `/proc/filesystems` entirely until the first instance, so a preflight that greps it reads the wrong thing. See `service/presentation/CLAUDE.md` for how the tier runs.

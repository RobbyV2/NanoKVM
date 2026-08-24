# bootslot

The userspace half of the A/B kernel scheme. The bootloader half lives in the kernel
fork under `scripts/ab/` (`uEnv.txt.in`, `mkuenv.sh`, `test-ab.sh`); the runtime half is
`kvmapp/system/init.d/S02abtrial`; the delivery half is
`server/service/application/kernel.go`.

## The bootloader contract

`boot.sd` is always the committed kernel and `boot.alt` always the trial. `uEnv.txt`
carries `ab_state=committed|trial` and redefines only `sdboot`, which is deliberately its
last line. `bootcnt` is four little-endian bytes, and `ab_limit=1`, so a trial gets a
single attempt before `ab_try` falls back to `ab_good_boot`.

**`uEnv.txt` must end with `\n\0`.** `env import` scans for that terminator; without it
u-boot parses up to a megabyte of whatever RAM follows the file. `setState` rewrites one
line in place and copies every other byte, terminator included, which is why nothing else
in this repository may write that file with its own serializer. `SetState` is the exported
entry point; use it rather than reimplementing the rewrite.

## The install order is the whole safety property

`installKernelPayload` runs five steps, and only this order survives losing power between
any two of them:

0. flip `ab_state` back to `committed`
1. write the new kernel over `/boot/boot.alt`, sync, re-read it and compare digests
2. zero `/boot/bootcnt`
3. flip `ab_state` to `trial` (temp file, rename, directory sync)
4. write `/etc/kvm/kernel_pending` with the version being tried

Until step 3 the bootloader still picks `boot.sd`, so a half-written `boot.alt` is never
selectable. Move the flip earlier and a torn kernel becomes bootable; that is the one
sequence that bricks the device, and
`TestKernelInstallSurvivesPowerLossAtEveryStep` fails on exactly that swap.

Step 0 exists because `ab_state` is not reset by a rollback: `ab_try` bumps `bootcnt` and
falls back to `ab_good_boot`, leaving the policy armed. A device that rolled back, or one
whose reboot never happened, therefore reaches the next install with `ab_state=trial`
already set, and writing into an armed slot is exactly the case the ordering exists to
prevent.

Step 1 writes straight over `boot.alt` with no temporary file because it cannot afford
one: **`/boot` holds under 2 MiB free against a ~7 MiB kernel**. The safe copy is the
extracted package in `CacheDir` on partition 2, which is why the read-back comparison is
against the package and not against a checksum carried alongside. Never create a third
file in `/boot`.

The pending marker and the confirmed kernel version live under `/etc/kvm`, not under
`/kvmapp`: an application update moves that whole tree to `/root/old` and puts a fresh
one in its place, so state kept there does not survive one.

## Committing is the other place /boot can run out

`Confirm` rewrites `boot.sd` in place from `boot.alt`, and `boot.sd` is also where a
rollback goes. An ENOSPC halfway through would leave the committed slot truncated and
nothing bootable, so `ensureRoomToCommit` compares the two sizes against the live free
space first and refuses. A refused commit costs one rollback and leaves the old kernel
whole, which is the correct trade. A kernel more than the free margin larger than the one
it replaces simply cannot be committed on this partition, and that is a packaging limit,
not a bug to work around here.

## A kernel package ships alone

A/B protects the kernel and nothing else. If one package carried both and the kernel
rolled back, the device would run the new application on the old kernel, a combination
nobody tested. `validateKernelPackage` therefore rejects any package whose root holds
anything besides `version` and `kernel/`, and `scripts/package.sh KERNEL_ITB=<boot.itb>`
is the producing half of the same rule. `kernel/boot.itb` must begin with the FDT magic
`d00dfeed`, because a payload u-boot cannot parse is only discovered by a device that no
longer boots.

## Confirming needs more than a listener

A kernel that boots with no working NIC reaches userspace perfectly well and leaves a
device nobody can reach. `ConfirmWhenReady` therefore waits for the UI to answer on the
loopback *and* for a non-loopback, non-link-local address to exist. Committing on the
listener alone would make that failure permanent.

## What `/dev/watchdog` actually does

The trial guard was designed around a claim read from source: that `dw_wdt` sets
`WDIOF_MAGICCLOSE`, so a close without writing `V` leaves the kernel petting the device
forever and the watchdog cannot notice the server dying. `scripts/kernelint.sh watchdog`
runs that claim in a VM against `softdog` and against `stopless_wdt`, a stub shaped like
`dw_wdt` with no reset control (its `stop()` only sets `WDOG_HW_RUNNING`, and it reports
a `max_hw_heartbeat_ms`). The measured result is the opposite:

| probe                                | softdog  | stopless_wdt |
| ------------------------------------ | -------- | ------------ |
| holder killed without writing `V`    | reboots  | reboots      |
| holder killed after writing `V`      | survives | survives     |
| holder frozen with SIGSTOP           | reboots  | reboots      |

Closing without the magic character is what keeps the watchdog armed
(`watchdog: watchdog0: watchdog did not stop!`), so a hardware watchdog *can* notice the
server dying. The "kernel pets it forever" behaviour is real but belongs to the other
branch: after `V` is written, the core calls `stop()`, `dw_wdt` cannot stop, and the
core's keepalive worker feeds the device indefinitely.

Nothing in this repository opens `/dev/watchdog`, so today the enabled `dw_wdt` provides
no protection at all and `S02abtrial` is the only trial deadline. That guard is still
needed; only its stated justification was wrong. Anything built on the watchdog later
must write `V` before every intentional restart, or an ordinary `S95nanokvm restart`
becomes a reboot.

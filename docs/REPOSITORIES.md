# Repositories, builds and device operation

## Workspace and ownership

| Repository | Branch | Upstream | Owns |
| --- | --- | --- | --- |
| `NanoKVM` | `main` | `sipeed/NanoKVM:main` | Go application, React UI, system scripts, native support and OTA packaging |
| `LicheeRV-Nano-Build` | `nanokvm-custom` | `sipeed/LicheeRV-Nano-Build:NanoKVM` | Kernel, U-Boot, OpenSBI, Buildroot and SD images |

The forks are under `RobbyV2` on GitHub. Merge only the upstream branch in this
table. The SDK's `upstream/main` targets MaixCAM and has a different rootfs and
boot configuration. Read its `CLAUDE.md` before editing or merging it.

`third_party/{wstunnel,newt,usb-proxy,hev-socks5-tunnel}` are submodules. Each fork
uses the `NanoKVM` branch and its project's upstream `main`. Read actual pins with
`git submodule status --recursive`; do not infer a pin from a document. The hev
fork contains nested submodules, so initialize with `--recursive`.

Additional sibling worktrees and firmware artifact directories are optional.
Use `git worktree list`, `git status`, and `git remote -v` to inspect the current
checkout. A directory of built firmware is not an independent build system.

## Source map

| Feature | Application code |
| --- | --- |
| Browser camera, microphone and speaker | `web/src/pages/desktop/menu/devices/`, `server/service/sources/`, `server/service/media/` |
| HDMI desktop streaming | `web/src/pages/desktop/screen/`, `server/service/stream/`, native vision library |
| Assistant gestures and requests | `web/src/pages/desktop/assistant/`, `server/service/assistant/` |
| Persistent USB presentation | `server/service/presentation/` |
| Browser WebUSB relay | `server/service/webusb/`, browser `devices/webusb.ts` |
| Exact/Hybrid passthrough | `server/service/passthrough/`, `server/service/functionfs/` |
| Network bridge | `server/service/bridge/` |
| Exit tunnel | `server/service/exit/`, `server/proto/exit.go`, `nexit/` |
| EDID | `server/service/edid/` |
| A/B kernel installation | `server/service/bootslot/`, `server/service/application/kernel.go`, SDK `scripts/ab/` |
| Virtual media and `/data` ownership | `server/service/storage/`, `web/src/pages/storage/`, presentation `SetLUN` and `DiskGuard` |
| Accounts, roles and permissions | `server/authn/`, `server/middleware/`, `server/service/auth/`, `server/router/` |

Per-package `CLAUDE.md` files define the relevant invariants. The browser webcam
path sends JPEG/NKMF into UVC; HDMI H.264 and WebUSB are independent paths.

## Builds and tests

Frontend requirements are Node >=22 and pnpm >=11. `corepack pnpm` can be used
where the standalone pnpm shim is absent. From `web`, run `pnpm test`, `pnpm
build`, and ESLint on changed files. The build includes TypeScript checking.

The RISC-V application uses the Docker builder and Sophgo's x86_64 host toolchain:

```sh
make builder-image
make check-image
make app DOCKER_TTY=
```

`DOCKER_TTY=` is required without an interactive terminal. The image name defaults
to `nanokvm-builder-local-<uid>-<gid>`; `IMAGE_NAME` selects an existing builder.
The image's `/home/build` ownership must match the build user's UID/GID. Rebuild
with the correct Docker build arguments when those identities differ.

| Target | Output |
| --- | --- |
| `make app` | `server/NanoKVM-Server` |
| `make web` | `web/dist/` |
| `make support` | `kvmapp/kvm_system/kvm_system` |
| `make vision` | `kvmapp/server/dl_lib/` |
| `make tunnels` | `kvmapp/tunnels/{wstunnel,newt}.gz` |
| `make passthrough` | `kvmapp/passthrough/usb-proxy.gz` |
| `make exit` | `kvmapp/exit/hev-socks5-tunnel.gz` |
| `make release-build` | All application RISC-V artifacts |
| `make release VERSION=x.y.z` | Application package and manifest |
| `make kernelint[-tier1\|-tier2\|-watchdog\|-zram]` | Kernel integration tests in QEMU |

The server requires cgo and the RISC-V `libkvm.so`. Setting `CGO_ENABLED=0`
drops `common.GetKvmVision`; it does not provide a portable application build.
Tests for independent packages can run on the host. The CI workflow derives that
package set from the import graph, runs Go tests/vet and kernel integration, and
builds/tests the frontend. Direct frame-queue tests are independent of libkvm:
`go test -race ./service/stream/direct/internal/framequeue` from `server`.

Cross-compiled C/C++ must carry the C906 flags on compile and link commands:
`-mcpu=c906fdv -march=rv64imafdcv0p7xthead -mcmodel=medany -mabi=lp64d`.
Verify the ELF interpreter and `$ORIGIN/dl_lib` RPATH against the device. Use the
Makefile/build scripts instead of a separate ad hoc application build.

Rust tunnel builds use `riscv64gc-unknown-linux-musl`, `rust-lld`, static CRT,
`uuid_unstable`, and release opt-level `z`. The older host-tools linker does not
accept every ISA attribute produced by Rust. Keep the Makefile's explicit feature
set; do not replace it with `--all-features`.

## Kernel builds

Use a case-sensitive SDK checkout. Linux headers include paths differing only by
case; a bind mount cannot repair files already collided on a case-insensitive
host. Never commit those filesystem artifacts. Check incoming paths against local
case collisions before merging the SDK.

```sh
docker build -t nanokvm-kbuild:22.04 -f docker/kbuild.Dockerfile docker
docker run --rm -v "$PWD":/work -w /work nanokvm-kbuild:22.04 \
  bash -c 'source build/cvisetup.sh && defconfig sg2002_licheervnano_sd && build_all'
```

The image supplies `/usr/local/host-tools`; the SDK's `host-tools` link must resolve
to it. Output lives under `install/soc_sg2002_licheervnano_sd/`. To assemble an SD
image, use `kvm/merge_nanokvm_app.sh` with the staged application tree and the SDK
builder's fuse support. Both script lists in that merge script must include every
required application init script.

A kernel-only package is produced by
`KERNEL_ITB=/path/to/boot.itb scripts/package.sh <version>` in NanoKVM. Follow the
A/B installation contract; `/boot` has limited space and cannot hold a third
kernel. Browser-only corrections do not require a kernel update.

## Upstream and release contracts

Preserve the fork's presentation, bridge, media, assistant, exit and packaging
behavior during upstream merges.

Accounts carry a role and upstream's permission grants (`authn/permissions.go`);
administrators hold every grant. Upstream routes use upstream's grants, except where
a grant would reach further than its catalog risk says: enabling SSH needs `terminal`
because it logs in as root, `tailscale` is rated `warn` because it can enroll the
device into the grantee's tailnet, and a URL download by a non-administrator may
reach public addresses only (`service/download/guard.go`). Keep these when merging
upstream's routes and catalog. Fork routes
whose feature has no grant in that catalog (presentation, passthrough, EDID, VNC,
favicon, bridge, tunnels, exit, assistant, media slots, kernel slots and startup
status) stay behind `RequireRole(authn.RoleAdmin)`, and their settings panels render
for administrators only. Browser media bindings follow the console grants:
`console.view` reads slot state and `console.control` binds a source. When an
upstream group's gate changes, move fork routes out of it rather than letting them
inherit the new gate. `S30rndis` resolves the live gadget NIC from
ConfigFS, supports NCM, matches DHCP daemons by config path and supports repeated
start after a rebind. Do not replace those contracts with hardcoded `usb0` logic.

The newt fork excludes Docker scanning, self-updating, telemetry, Prometheus,
OpenTelemetry and pprof. Keep those deletions when merging upstream and keep
`NEWT_VERSION` aligned with the fork's base tag. Build Go dependencies with the
version supplied by the builder. Test usb-proxy changes in its own repository
and with the relevant kernel integration tier.

After a submodule changes, rebuild its compressed seed. Seeds are gitignored;
a fresh clone needs those builds before packaging. The package script checks
artifact architecture, freshness, RPATH and compressed/uncompressed equivalence.
Use `gzip -9 -n` for reproducible seeds.

`scripts/README.md` owns the release contract. `latest.json` stores SHA-512 as
base64 of the raw digest. Release archives use GNU tar and `SOURCE_DATE_EPOCH`.
Application and kernel payloads are separate so an A/B rollback cannot leave an
untested application/kernel combination. CDN publication is a separate operation;
building or pushing source does not offer firmware to devices.

## Runtime paths and service lifetime

| Path | Purpose |
| --- | --- |
| `/kvmapp/server` | Persistent application binary, libraries and web assets |
| `/tmp/server` | Running application tree copied by `S95nanokvm` |
| `/etc/kvm` | Persistent configuration and runtime-managed application state |
| `/etc/kvm/bin` | Extracted managed binaries |
| `/root` | Deployment staging, rollback archives and diagnostic output |
| `/boot` | Bootloader policy and A/B kernel slots |

An application OTA replaces `/kvmapp`; settings that must survive belong under
`/etc/kvm`. `/tmp` is a small shared tmpfs used by the application, DHCP and other
services. `S95nanokvm` copies the whole persistent server directory, so backup
binaries or web trees beside the server consume tmpfs on every restart. Keep
staging and backups under `/root`, and inspect `df -h /tmp` before deployment.

Restart over SSH with independent standard streams:

```sh
/etc/init.d/S95nanokvm restart > /root/nanokvm-restart.log 2>&1 < /dev/null
```

Leaving server stdout/stderr attached to the SSH pipe can keep the session open
and terminate the server with SIGPIPE when it logs after the connection closes.
Inspect `/proc/<pid>/fd/1` and `/proc/<pid>/fd/2` to locate current output. UVC and
startup traces also appear in `dmesg`.

When JWT's configured signing key is empty, the server generates an in-memory
key at startup. Restarting invalidates old cookies; fresh login distinguishes
session expiry from media/network failure. WebSocket 4401 means expired/revoked
session. A generic 1006 needs its underlying browser or network error.

## Deploying application changes

For a web-only update, build and back up both web trees under `/root`, install new
assets in `/kvmapp/server/web` and `/tmp/server/web`, then rename the new
`index.html` into place last. Keep previous hashed assets while open tabs can
still request them. The static server needs no restart. Verify hashes and the
served index, then hard-refresh an isolated browser and test the deployed page.

For a server update, stage the binary under `/root`, verify architecture/library
requirements, back up the persistent binary there, and replace the persistent
binary by rename before restarting with the command above. Confirm the new PID,
served assets, authenticated API, device state and media output after restart.
Preserve configuration, the USB profile, native libraries and the custom kernel
unless the change specifically requires updating them.

The device has BusyBox utilities. Its `tar` supports plain `-cf`/`-xf` and may
reject `-z`; use a plain host-created tar instead of assuming GNU options. The
installed SCP server can require the client's legacy mode (`scp -O`).

## Boot and USB contracts

Boot scripts run in SysV order. `S01fs` mounts ConfigFS/debugfs, `S02abtrial` guards
a kernel trial, `S02zram` configures swap, `S03usbdev` prepares the gadget and
`S95nanokvm` starts the application. The presentation manager binds the UDC after
its functions and media event subscriptions are ready. Never reorder binding
ahead of the UVC subscription: this fork forwards UVC class requests to userspace.

The virtual disk serves the data partition the device also mounts at `/data`, and
only one side may write it. `S01fs` mounts `/data` read-only when `/boot/usb.disk0`
says the disk serves the raw partition writable. At runtime the presentation manager
owns the mass-storage LUN: `service/storage` changes the medium only through
`SetLUN`, and its partition guard makes `/data` read-only before the partition is
served writable and writable again once it is not, after every gadget change.

`new_app_init()` copies a fixed init-script list, SDK image assembly has two
script lists, and extensions can install their own seeds at service startup.
Choose the correct delivery mechanism. App initialization removes `S99*`; new
scripts must use another supported slot. Do not assume a shipped seed is installed.

Read the live network function's `ifname` rather than assuming `usb0`. A profile
apply can re-enumerate both camera and network functions. Read active profile
contents and ConfigFS links; names do not imply which functions are enabled.

The custom SDK supplies bridge, zram, media/controller and exit-routing support.
Check `/proc/config.gz` for deployed capabilities. Minimal images have a smaller
utility/package set; commands available in the builder are not necessarily on the
device. Physical microSD recovery is required if no kernel boots.

# wstunnel and newt integration

Status: built and committed. Both binaries cross-build and link, `make tunnels` completes and produces both seeds, the Go tunnel package tests pass and the web build is green. Nothing has run on real hardware.
Date: 2026-08-21.

## Goal

Ship two tunnel clients as first-class NanoKVM features alongside Tailscale: `wstunnel` (erebe/wstunnel, Rust, WebSocket and HTTP2 tunnel) and `newt` (fosrl/newt, Go, connector for a Pangolin reverse proxy). Both are baked into the firmware, both are replaceable on a running device, and both are configured through a raw argument string plus an environment table rather than a bespoke form.

## Why the Tailscale pattern does not transfer whole

Tailscale's page works because `tailscale status --json` is a state oracle. The daemon owns its own state and NanoKVM only asks.

| | state source | config source |
|---|---|---|
| tailscale | `tailscale status --json` | daemon's own `tailscaled.state` |
| wstunnel | none: pid and log tail only | none: CLI flags only |
| newt | none: `--health-file` and `:2112` metrics | its own `CONFIG_FILE` JSON |

`wstunnel-cli/src/main.rs` is a bare `clap::parse()`. There is no config file for CLI options and no `--config` flag, and only eight settings carry an env var (`HTTP_PROXY`, `WSTUNNEL_HTTP_PROXY_LOGIN`, `WSTUNNEL_HTTP_PROXY_PASSWORD`, `WSTUNNEL_HTTP_UPGRADE_PATH_PREFIX`, `WSTUNNEL_RESTRICT_HTTP_UPGRADE_PATH_PREFIX`, `RUST_LOG`, `NO_COLOR`, `TOKIO_WORKER_THREADS`). An argument string plus an env table is therefore the complete configuration surface, not a shortcut around one. newt is the mirror image: all of its roughly forty settings have an env var, so an env table alone configures it fully.

Two consequences drive the design. NanoKVM owns the config and renders an argv from it. Status is inferred from a pid check and a liveness probe rather than queried, and absence or failure is always a state value inside a `code: 0` response, never an error code.

## Decisions

| # | Decision |
|---|---|
| D1 | wstunnel supports client and server invocations. Server mode with neither `--restrict-config` nor `-r` is an open forward proxy into the LAN, flagged with a warning icon and a few words under the argument field. No modal, no block. |
| D2 | Binaries ship gzipped at `/kvmapp/tunnels/<name>.gz` and run from `/etc/kvm/bin/<name>`, extracted on first start. One binary path, not two. |
| D3 | newt credentials go in the env table, seeded with empty `PANGOLIN_ENDPOINT`, `NEWT_ID`, `NEWT_SECRET` and `NEWT_PROVISIONING_KEY` rows. |
| D4 | Config persists at `/etc/kvm/{wstunnel,newt}.json`, mode 0600, plaintext. The SD card is trusted storage. |
| D5 | Enabled equals autostart equals init script present, matching Tailscale exactly. No separate boot checkbox. |
| D6 | No `system_init.cpp` change. The Go service copies its init script on start and on server boot. |
| D7 | `start-stop-daemon` for supervision, plus a 30 second watchdog in the Go server that re-runs start when an enabled service's pid is gone. |
| D8 | newt's memory is bounded by exporting `GOMEMLIMIT` from the existing `/etc/kvm/GOMEMLIMIT` plus `GOGC=50`. No second knob. |
| D9 | All combinations of tailscale, newt and wstunnel are allowed. A one-line notice appears when more than one runs. |
| D10 | newt is cut down on the fork branch: self-update, Docker socket scanner, OTel, Prometheus and pprof are removed. |
| D11 | wstunnel ships at `opt-level = "z"`: 5,250,408 bytes against 7,591,328 at `opt-level` 3, and 2,730,864 against 3,999,977 gzipped. `ring`'s crypto is C compiled by the `cc` crate at its own `-O2` regardless, so `z` only shrinks the async and IO glue. |
| D12 | Seeds are gzipped, decompressed in-process with `compress/gzip`. No new dependency. squashfs is not used: the kernel is built out of tree with no source, config or defconfig in this repo, so enabling it would mean forking the SDK and shipping a divergent base image. |
| D13 | The tunnel binaries never run from `/tmp`. That is tmpfs, and `S95nanokvm` already copies the server, its 7.65 MB of shared objects and the whole web bundle there. |

## Submodules and fork branches

| Path | Fork | Upstream pin | Branch |
|---|---|---|---|
| `third_party/wstunnel` | `RobbyV2/wstunnel` | v10.6.2 | `NanoKVM` |
| `third_party/newt` | `RobbyV2/newt` | v1.16.0 | `NanoKVM` |

Both are submodules now, alongside `third_party/usb-proxy` for the passthrough seed. `third_party/wstunnel` sits at `v10.6.2-2-g9b38f18`: two fork commits past the tag, still the v10.6.2 upstream pin.

**wstunnel branch.** No source patch is required. riscv64 needs only `--no-default-features --features ring`, a configuration upstream already exercises for armv7, armv6, freebsd-x86 and windows-x86, and CI runs the full test suite under it. The branch carries a riscv64 entry in `.github/workflows/release.yaml`, worth offering upstream as a pull request.

Pin to the v10.6.2 tag, not `main`: the fork branch carries its own commits on top of the tag rather than tracking main, and refreshing the pin means moving those commits, not moving the base. The rendered README documents unreleased main-branch CLI including `--enable-webtransport`, the `wts://` scheme, `--websocket-ping-frequency <DURATION>` in place of the released `--websocket-ping-frequency-sec <seconds>`, `--remote-to-local-server-idle-timeout` and `--dns-resolver-prefer-ipv4`. Code written against the README produces flags the shipped binary rejects.

**newt branch.** Remove the self-update goroutine at `main.go:184-213`. Two minutes after start and every six hours after, it fetches the latest build for `linux_riscv64`, verifies SHA-256, renames over the running binary and re-execs. The platform map includes riscv64, `/usr/bin` and `/kvmapp` are both writable, so left alone it replaces a stripped 12.8 MB build with upstream's 35.5 MB one, unattended, on flash. `NEWT_SYSTEM_SUBSTRATE=CONTAINER` is also set in the generated wrapper, so the two mitigations fail independently.

Then remove the Docker socket scanner, the OTel exporter, the Prometheus client and pprof, along with the flags and config keys that reach them: `--docker-socket`, `--docker-enforce-network-validation`, `--metrics`, `--otlp`, `--metrics-admin-addr`, `--metrics-async-bytes`, `--pprof`. None is reachable on a NanoKVM. The auth daemon, PAM and the native TUN path stay, because native mode is the fallback if the userspace netstack underperforms.

## Build

`docker/Dockerfile` gains a rust layer. The builder image already carries `riscv64-unknown-linux-musl-gcc` from the Sophgo host-tools tarball and has no rustup or cargo. wstunnel needs Rust 1.85 or newer, since `edition = "2024"` with `resolver = "3"` sets that floor; upstream CI pins 1.97 and there is no `rust-toolchain.toml` or `rust-version` field.

A `make tunnels` target cross-builds both into `kvmapp/tunnels/`, which is gitignored. `scripts/package.sh` already sweeps that directory up through `cp -a "$ROOT/kvmapp/."`, so packaging needs only two changes: `package` depends on `tunnels`, and the existing `require_riscv64()` check (`od -An -tu1 -j18 -N1`, asserting `e_machine == 243`) runs on both outputs.

wstunnel:

```
rustup target add riscv64gc-unknown-linux-musl
export CC_riscv64gc_unknown_linux_musl=riscv64-unknown-linux-musl-gcc
export CARGO_TARGET_RISCV64GC_UNKNOWN_LINUX_MUSL_LINKER="$(rustc --print sysroot)/lib/rustlib/$HOST/bin/rust-lld"
cargo build --release --bin wstunnel \
  --target riscv64gc-unknown-linux-musl \
  --no-default-features --features ring
```

Three constraints. `.cargo/config.toml` sets `rustflags = ["--cfg", "uuid_unstable"]` and uuid v7 will not compile without it. Never `--all-features`, because it enables both `aws-lc-rs` and `ring`, whose jsonwebtoken providers panic at runtime. Leave jemalloc off, since upstream enables it only on 64-bit targets and it needs `JEMALLOC_SYS_WITH_LG_PAGE` tuned to the page size.

wstunnel must link with `rust-lld`, not with the `riscv64-unknown-linux-musl-gcc` driver. The Sophgo host-tools carry GNU ld 2.35 from 2020, and Rust 1.97's LLVM stamps a modern ISA string into `.riscv.attributes` naming `zifencei`, `zmmul`, `zaamo`, `zalrsc`, `zca` and `zcd`. Those extension names postdate that linker, so it aborts the attribute merge with `Invalid or unknown z ISA extension: 'zifencei'`, on rustc's own objects and on the prebuilt `libcompiler_builtins` that ships with the rustup target. The symptom reads like a corrupt object file rather than a linker five years behind the ISA naming, so it is worth recognising. `ring`'s C still compiles through gcc 10.2 unchanged; only the link moves. The Makefile resolves `rust-lld` at container runtime from `rustc --print sysroot` plus the host triple from `rustc -vV`, so a toolchain bump does not break it, and adds `-C target-feature=+crt-static`, which this target does not set by default. `-C link-self-contained=yes` is redundant, since musl infers self-containment from crt-static; builds with and without it are byte-identical. `-C linker-flavor` is unnecessary because rustc infers `gnu` from the `rust-lld` stem. Nothing else in the repository feeds Rust objects to that linker, so the Go and C++ targets are unaffected and binutils 2.35 can stay.

`aws-lc-rs` is the default provider and cannot be used here. `aws-lc-sys` ships pregenerated bindings for a fixed target list that excludes `riscv64gc-unknown-linux-musl`, and upstream closed that request with gnu-only support. `ring` has no pregenerated bindings and no cmake dependency, and its `src/cpu/` covers only aarch64, arm, x86 and x86_64, so riscv64 needs no feature detection and gets no crypto assembly.

The shipped binary is built at `opt-level = "z"`. Upstream's release profile already sets `lto = "fat"`, `panic = "abort"`, `codegen-units = 1` and `strip = "symbols"`, so `opt-level` is the only lever left, and it applies from the command line, which leaves the submodule's `Cargo.toml` untouched:

```
cargo --config 'profile.release.opt-level="z"' build --release ...
```

newt:

```
CGO_ENABLED=0 GOOS=linux GOARCH=riscv64 go build \
  -trimpath -ldflags="-s -w -X main.newtVersion=$VER -X main.newtPlatform=linux_riscv64" \
  -o newt .
```

Go's riscv64 ABI is lp64d and needs hardware double-precision floating point, which the SG2002 C906 has. `riscv64gc` is a compatible baseline against the SoC's RV64IMAFDCV; neither build uses the T-Head vector 0.7 extension that NanoKVM's C and Go builds target with `-march=rv64imafdcv0p7xthead`.

Sizes, measured from builds of both fork branches:

| Binary | Upstream | Built | Gzipped seed |
|---|---|---|---|
| newt | 35.5 MB unstripped | 12,845,240 | 4,993,439 |
| wstunnel | 9.5 MB (armv7 ring build: 8.5 MB) | 5,250,408 | 2,730,864 |

Every figure is measured from a completed `make tunnels`. newt is built with the go1.25.0 the Dockerfile pins, matching the `go 1.25.0` in the server's own `go.mod`; building it with go1.26.5 instead gives 12,320,930, and the same toolchain rebuilding the pre-strip commit gives the same size, so the subsystem removal is what shrank it rather than the compiler. wstunnel was also cross-built natively on darwin/arm64 as a cross-check and came out within 1,512 bytes raw and 2,898 bytes gzipped of the container build, across a different linker and a different host.

`make tunnels` gzips each binary with `gzip -9 -n` into `kvmapp/tunnels/<name>.gz`. The `-n` matters: `scripts/package.sh` already derives `SOURCE_DATE_EPOCH` from the commit date and pipes through `gzip -n -9` for reproducibility, and an embedded timestamp would break that.

`.github/workflows/package.yml` gains a `make tunnels` step with a cargo registry and target cache. Submodules need `submodules: recursive` on the checkout.

## Binary storage and replacement

One path: the service runs `/etc/kvm/bin/<name>`. When that file is absent, the Go server extracts it from `/kvmapp/tunnels/<name>.gz` with `compress/gzip`, writes it atomically with `os.CreateTemp` and `os.Rename` in the same directory, and chmods it 0755.

`/kvmapp` is replaced wholesale on every OTA by `server/service/application/install.go`, which moves the old tree to `/root/old` and moves the new one into place. `/etc/kvm` is never touched by the updater. So the gzipped seed refreshes with each firmware update and the extracted binary persists, which means a firmware upgrade does not silently replace a binary the user put there deliberately. Deleting the extracted file reverts to the shipped version on the next start.

Storage is not scarce here. `S01fs` grows the rootfs partition to 8192 MB on first boot and the smallest hardware variant is an 8 G eMMC, so the updater's `max(5%, 128 MiB)` reserve resolves to 400 MB rather than 128. Compression is worth doing anyway because everything in `/kvmapp` is effectively tripled during an update, between the downloaded tarball, the extracted tree and the backup at `/root/old`. Gzipped seeds total 7,724,303 bytes against 18,095,648 uncompressed, so about 9.9 MB of flash and about 30 MB of transient update space. newt compresses to 38.9 percent of its size and wstunnel to 52.0 percent.

gzip is the only format already present on the device. `server/utils/untar.go` uses `compress/gzip` and `archive/tar`, `server/service/application/archive.go` decompresses the OTA payload in-process, and `server/service/picoclaw/runtime_install.go` does the same for its own download. There is no xz, zstd, lz4, brotli or snappy anywhere in the dependency graph, direct or indirect, and no evidence that any busybox compression applet exists on the device rootfs, which is out of tree.

squashfs would be better still, mounting both binaries read-only and executing in place so that no extracted copy exists and only touched pages cost RAM. The repo has zero references to squashfs, loop devices or overlayfs, and no kernel config is committed, so it cannot be designed for. `cat /proc/filesystems | grep squashfs` and `ls /dev/loop*` on a running device settle it; if both are positive this becomes a clean swap for the extraction step.

`POST /api/extensions/tunnel/:name/binary` accepts a multipart upload and writes `/etc/kvm/bin/<name>` directly, mirroring `server/service/application/update_offline.go`: an `X-SHA256-Checksum` header, a filename matched against `^[a-zA-Z0-9._-]+$`, and a size cap. It asserts `e_machine == 243` on the uploaded ELF before installing it, and refuses while the service is running. `DELETE` on the same path removes the file, so the next start re-extracts the shipped seed.

The binaries never run from `/tmp`. `S95nanokvm` copies `/kvmapp/kvm_system` and `/kvmapp/server` there wholesale, and `/kvmapp/server` carries 7.65 MB of shared objects plus the entire built web bundle. All of that is tmpfs, which is RAM, on a device where RAM is the binding constraint and newt's footprint is the top risk.

## Config and lifecycle

Config lives at `/etc/kvm/wstunnel.json` and `/etc/kvm/newt.json`, mode 0600, written atomically with `os.CreateTemp` in the same directory followed by `os.Rename`, cloned in shape from `server/service/mcp/config.go` including the package-level mutex and the injectable `configFilePath` var that makes it testable.

```go
type Config struct {
	Args string            `json:"args"`
	Env  map[string]string `json:"env"`
}
```

Nothing else is stored. Enabled state is the presence of `/etc/init.d/S97<name>`, which is how Tailscale already works.

On save and on start, Go tokenizes `Args` with shell quoting rules and writes a generated wrapper at `/etc/kvm/<name>.cmd`, mode 0700:

```sh
#!/bin/sh
export PANGOLIN_ENDPOINT='https://pangolin.example.com'
export NEWT_SYSTEM_SUBSTRATE='CONTAINER'
export GOMEMLIMIT='48MiB'
export GOGC='50'
exec /etc/kvm/bin/newt '--health-file' '/tmp/newt.health' '--config-file' '/etc/kvm/newt-client.json'
```

Go controls every quote, so there is no `eval`, no interpolation of user text into an `sh -c` string, and no injection surface. Tailscale's `exec.Command("sh", "-c", cmd)` is safe only because every command it builds is a literal. Tokenization doubles as validation: unbalanced quotes fail the save with a message instead of producing a service that will not start.

The init script is then fixed at about fifteen lines and never needs regenerating:

```sh
start-stop-daemon -S -bmq -p /var/run/<name>.pid -x /etc/kvm/<name>.cmd
```

`S97` is free. `S98tailscaled` and `S96picoclaw` bracket it, and `support/sg2002/kvm_system/main/lib/system_init/system_init.cpp:127` prunes `S99*` to speed up boot, so that range is unusable.

Per-service settings that NanoKVM injects rather than leaving to the user:

| | injected env | injected args |
|---|---|---|
| wstunnel | none | none |
| newt | `NEWT_SYSTEM_SUBSTRATE=CONTAINER`, `GOMEMLIMIT`, `GOGC=50` | `--health-file /tmp/newt.health`, `--config-file /etc/kvm/newt-client.json` |

`GOMEMLIMIT` comes from the existing `/etc/kvm/GOMEMLIMIT` file, a plain integer in MiB shared with `GET|POST /api/vm/memory/limit` and already exported into tailscaled's environment by `S98tailscaled`. `GOGC=50` is the lever that actually reduces resident memory; a limit alone sets a ceiling without pulling RSS down.

newt's own `CONFIG_FILE` is deliberately separate from NanoKVM's config. newt rewrites that file during provisioning-key registration, storing the returned `newtId` and `secret` and clearing the key, so it must be writable and must not be a file NanoKVM also writes. newt calls `MkdirAll` on the default config directory at startup whether or not it is used, which is the other reason to point it somewhere sane.

## Status

Inferred, in order:

1. Binary missing at both paths gives `notInstall`.
2. `Args` empty for wstunnel, or no endpoint and no credentials in the env for newt, gives `notConfigured`.
3. No live pid gives `stopped`. Liveness is the pidfile plus a `/proc/<pid>/cmdline` basename check, the idiom in `S96picoclaw`'s `is_gateway_pid()`.
4. Live pid gives `running`.
5. For newt only, `/tmp/newt.health` present promotes `running` to `connected`. newt writes the literal bytes `ok` at mode 0644 when the tunnel is healthy and removes the file when it drops or shuts down, so file existence is the probe.
6. A live pid with errors in the recent log gives `error`, carrying the log tail as `message`.

wstunnel stops at `running`, because there is genuinely nothing else to read. Logs go to `/tmp/<name>.log` on tmpfs, matching `/tmp/picoclaw.log`.

## API

One package serving two instances, with the service name as a route parameter.

```
GET    /api/extensions/tunnel/:name/status
GET    /api/extensions/tunnel/:name/config
POST   /api/extensions/tunnel/:name/config
POST   /api/extensions/tunnel/:name/start
POST   /api/extensions/tunnel/:name/stop
POST   /api/extensions/tunnel/:name/restart
GET    /api/extensions/tunnel/:name/logs
POST   /api/extensions/tunnel/:name/binary
DELETE /api/extensions/tunnel/:name/binary
```

`:name` is validated against the `Name` enum and anything else is rejected. Routes go in the existing admin group in `server/router/extensions.go`, which already carries `CheckToken()` and `RequireRole(authn.RoleAdmin)`.

`GET /config` never returns a secret. Env values whose key matches a secret pattern come back as a `configured` boolean plus the last four characters. Tailscale sets no precedent here because it never handles one.

## Files

Backend:

| Path | Action |
|---|---|
| `server/proto/tunnel.go` | new: `Name` enum, `State` enum, request and response types |
| `server/service/extensions/tunnel/service.go` | new: gin handlers, `NewService(name)` |
| `server/service/extensions/tunnel/config.go` | new: atomic load and save |
| `server/service/extensions/tunnel/wrapper.go` | new: tokenize, quote, render `.cmd`, resolve binary path |
| `server/service/extensions/tunnel/status.go` | new: pid and health inference, log tail, watchdog |
| `server/router/extensions.go` | modify: two instances, nine routes each |

Per-service differences live in one table keyed by `Name`, covering the injected env, injected args, seeded env keys and health probe. There is no second package.

Device and build:

| Path | Action |
|---|---|
| `kvmapp/system/init.d/S97wstunnel` | new |
| `kvmapp/system/init.d/S97newt` | new |
| `.gitmodules` | new |
| `docker/Dockerfile` | modify: rust layer |
| `Makefile` | modify: `tunnels` target, `package` depends on it |
| `scripts/package.sh` | modify: `require_riscv64` on both binaries |
| `.gitignore` | modify: `kvmapp/tunnels/*.gz` |
| `.github/workflows/package.yml` | modify: recursive submodules, `make tunnels`, cargo cache |

Frontend:

| Path | Action |
|---|---|
| `web/src/api/extensions/tunnel.ts` | new: one client, name in the path |
| `web/src/pages/desktop/menu/settings/tunnel/index.tsx` | new: the shared page |
| `web/src/pages/desktop/menu/settings/tunnel/env.tsx` | new: key and value table |
| `web/src/pages/desktop/menu/settings/tunnel/binary.tsx` | new: upload and revert |
| `web/src/components/icons/newt.tsx` | new |
| `web/src/pages/desktop/menu/settings/index.tsx` | modify: two tab entries |
| `web/src/i18n/locales/*.ts` | modify: all 24 |

Both pages are the same component, taking `service: 'wstunnel' | 'newt'` and registered twice in the `tabs` array inside the existing `isAdmin` spread. Four new files rather than the sixteen a literal Tailscale clone produces.

## Page

Header with state and start, stop and restart. An argument textarea. An env key and value table, seeded for newt with four empty credential rows. A log tail. The binary override control showing which path is in use.

The wstunnel server warning is a warning icon and a few words directly under the argument field, shown when the argument string begins with `server` and contains neither `--restrict-config` nor `-r`. Without a restriction file, wstunnel builds its rules from `RestrictionsRules::from_path_prefix()`, whose own comment reads that if no path prefixes are provided it allows all, making a bare `wstunnel server wss://[::]:8080` a forward proxy to the internet and to the LAN behind it.

Conventions that break the build if missed: named arrow-function exports with no default export, `.ts` extensions on `@/` imports, a re-entrancy guard of `if (isLoading) return` on every handler, `.then` and `.catch` and `.finally` rather than async, errors into a local `errMsg` rendered in `text-red-500`, refresh through an `onSuccess` prop with no cache layer, and import ordering enforced by `@ianvs/prettier-plugin-sort-imports`. TypeScript runs with `noUnusedLocals` and `noUnusedParameters`, and `pnpm build` is `tsc && vite build`, so a dead import fails the build.

Widgets come from antd v5 under `theme.darkAlgorithm`, layout from Tailwind v3 with `preflight` disabled, scroll areas from the Radix wrapper at `web/src/components/ui/scroll-area.tsx`.

## Icons

The app is dark-only, with `theme.darkAlgorithm` hardcoded in `web/src/main.tsx` and no light branch. Every icon except Tailscale's is a single path inheriting `currentColor`, tinted with Tailwind text classes. Black and white therefore resolves to one file, not two.

wstunnel uses lucide's existing `CoffeeIcon`, so it costs no new file. `lucide-react` is already a dependency and is the repo's default icon mechanism.

newt has no logo of its own anywhere in the fosrl org. The only image in the repo is a screenshot, and the official Unraid template points at the shared Pangolin mark. `fosrl/pangolin` `public/logo/pangolin_black.svg` is 2,509 bytes, one `<path>`, one solid fill, no gradients or strokes or clip paths, with a `viewBox` of `0 0 238.34422 252.7317` and a `translate(-13.119542,-5.9258171)` on the content group. It converts cleanly to an `IconBase` component after running SVGO, normalizing the viewBox and replacing the fill with `currentColor`. The eye is a knockout hole in the path, so it reads correctly light-on-dark.

That mark is Fossorial's branding shipped inside an AGPLv3 repository, and the code license does not obviously grant logo use.

## i18n

Twenty-four locales exist and all carry a full `tailscale` block: ca, cz, da, de, en, es, fr, hu, id, it, ja, ko, nb, nl, pl, pt_br, ru, se, th, tr, uk, vi, zh_tw, zh. The loader uses `import.meta.glob('./locales/*.ts', { eager: true })`, so no registry edit is needed.

Tab labels are derived as `t('settings.${tab.id}.title')`, so `settings.wstunnel.title` and `settings.newt.title` are mandatory in every locale or the sidebar renders a raw key. Shared strings live under `settings.tunnel.*`. Each feature block redefines its own `okBtn` and `cancelBtn` rather than sharing a global, matching the existing convention.

All 24 locales get real translations, not English placeholders.

## Risks

Blocking, and taken in this order so failures surface early:

1. Resolved. wstunnel compiles for `riscv64gc-unknown-linux-musl` with `ring`, verified twice, and the output is `ELF 64-bit LSB executable, UCB RISC-V, RVC, double-float ABI, statically linked, stripped`. What the build exposed is that `ring` 0.17.14 contains no RISC-V code whatsoever: no assembly, no `AsmTarget` entry, no `cpu` module, and no `compile_error!` guard for unrecognised architectures, so riscv64 falls silently through to the portable C path. That includes `aes_nohw.c`, bitsliced constant-time software AES. ChaCha20-Poly1305 was already the likely winner without AES hardware; against `aes_nohw` it is not close. If throughput disappoints, restricting the offered ciphersuites is the first lever. A successful compile is not evidence of correct execution: nothing has run a RISC-V instruction yet.
2. Whether the newt riscv64 binary runs on this kernel. It is static lp64d and should, though no riscv64 CI job or user report exists upstream and the gvisor netstack may assume epoll or timerfd behavior this Buildroot kernel does not provide. First check on device is `newt --version` and `newt --show-config`.
3. newt's resident memory. The device has 256 MB with 158 MB reserved for the multimedia subsystem, leaving roughly 90 MB. newt's own Helm chart requests 128 Mi and limits 256 Mi. Measure RSS idle and with one proxied target under live video, with `GOGC=50` and `GOMEMLIMIT` in place.

High:

4. Crypto cost. `ring` on riscv64 has no crypto assembly, so a roughly 1 GHz C906 with no crypto extensions runs AEAD over the whole KVM stream as portable C. Benchmark ChaCha20-Poly1305 against AES-GCM; ChaCha should win without AES hardware. The same question applies to newt's userspace WireGuard.
5. newt throughput. Upstream discussion #512 documents 4 to 10 MB/s through the userspace proxy, traced by the maintainer to sequential one-packet-at-a-time processing, with the `netstack2` rewrite as the in-progress fix. Adequate for a few Mb/s of KVM stream; verify against the real 1080p bitrate.
6. Free space, now the mildest of these. `S01fs` grows the rootfs to 8192 MB and the reserve resolves to 400 MB on that size, against about 7.4 MB of gzipped seeds. Still run `df` after first boot and confirm an OTA clears the preflight threshold, since nothing in the repo documents real-world occupancy.

Medium:

7. `start-stop-daemon -b` against a static Go binary with a runtime thread pool. Confirm the pidfile records the right pid through the wrapper's `exec`, and that `-K` actually terminates it. picoclaw's elaborate `/proc/<pid>/cmdline` verification exists for a reason.
8. Whether `--config-file` genuinely overrides newt's default `MkdirAll` path, and whether `/etc/kvm` is writable at `S97` time rather than racing `S01fs`.
9. Boot ordering. `S97` runs after `S30eth` and `S30wifi`, though Wi-Fi association may not be complete. Confirm both services retry rather than exit when DNS fails at boot. wstunnel's reverse-tunnel backoff controls are main-branch only and absent from 10.6.2, so its cold-boot retry behavior needs empirical confirmation.
10. Self-update suppression. Leave the device running past two hours and confirm the newt binary's mtime and size are unchanged.
11. Extraction. Confirm the gzip seed extracts correctly on a slow core, that a partial or interrupted extraction leaves no executable behind (atomic temp and rename), and that the extracted binary survives an OTA while the seed is refreshed underneath it.

Low:

12. wstunnel needs the user to run a relay VPS, and the relay exposes a raw TCP port rather than an HTTPS vhost, so a friendly hostname needs nginx or Caddy in front. newt needs a Pangolin instance. Both are a higher operational bar than Tailscale's hosted coordination.
13. wstunnel's `-P` path-prefix secret travels in the URL path and lands in any reverse proxy or CDN access log. The `-H 'Authorization: Bearer ...'` scheme with a server-side `!Authorization` regex is cleaner, and mTLS is the only strong option.
14. The Pangolin trademark question above.

## Testing

`config.go`, `wrapper.go` and the seed extraction are pure enough to test and get table-driven tests, mirroring `server/service/mcp/config_test.go` with an injected `configFilePath` and `t.TempDir()`. Tokenization tests cover quoting, escapes, unbalanced quotes and injection attempts. Wrapper rendering tests assert that a value containing a single quote cannot escape its quoting.

Nothing in this repo mocks `exec.Command`, and no CI workflow runs `go test`, `go vet` or `pnpm lint`, so these tests will not run automatically. `make web` does run `tsc && vite build`, so TypeScript errors do fail the build.

# Notes for agents working in this repository

This repository is a fork of `sipeed/NanoKVM`. Upstream is the source of truth for
everything we have not deliberately changed, and we periodically merge
`https://github.com/sipeed/NanoKVM` `main` into our `main`.

## Temporary local patches that should be reverted when upstream fixes them

Some of our changes exist only because upstream is currently broken. These are not
improvements we want to own — they are stopgaps. When upstream ships its own fix,
drop ours and take upstream's, so we do not carry a permanent divergence that has
to be re-resolved on every merge.

### Virtual keyboard: CommonJS default-import unwrap

- **File:** `web/src/pages/desktop/virtual-keyboard/index.tsx`
- **Our commit:** `2a00b82` "unwrap the keyboard cjs default"
- **Upstream commit that broke it:** `761b02e` *feat(web): upgrade build tooling to
  Vite 8 (#853)*

`react-simple-keyboard` ships a UMD CommonJS build. Through Vite 7, the import site
received interop that unwrapped the module's `__esModule` default, so
`import Keyboard from 'react-simple-keyboard'` was the component. Vite 8 stopped
injecting that interop, so the default import now arrives as the raw
`module.exports` object (`{ KeyboardReact, default }`). Rendering that object throws
React error #130 ("expected a class/function but got: object"), and because the
error escapes to react-router's error boundary it takes the entire desktop route
down, not just the keyboard.

This was verified by bisection rather than inference: upstream at `761b02e^`
(Vite 7.3.2) renders the virtual keyboard correctly, while both upstream `main` and
our `main` (Vite 8.2.0) throw. The bug is therefore present in the pristine upstream
tree — we did not introduce it, we inherited it by merging. Note that
`optimizeDeps.needsInterop` does **not** fix it; that was tried and reverted.

Our patch renames the default import and unwraps `.default` when it is present, so
it is correct under either interop. **Delete it and restore the plain
`import Keyboard from 'react-simple-keyboard'` as soon as upstream resolves the
Vite 8 interop problem**, either by fixing the import themselves or by moving to a
Vite release that restores the behaviour.

## Workspace and sibling repositories

Full detail — workspace layout, cloning, building, per-repo merge procedures,
device runtime facts and known gotchas — is in `docs/REPOSITORIES.md`. Read it
before touching anything outside this repo. The essentials:

- Repos live side by side under a workspace root (`<workspace>/`, which on the
  author's Mac is `~/Documents/apcs`): `NanoKVM` (this repo), the kernel SDK fork
  `LicheeRV-Nano-Build`, the non-git artifact drop `nanokvm-firmware`, and six
  `git worktree` checkouts of this repo named `NanoKVM-*`.
- Merge `upstream/main` here. In `LicheeRV-Nano-Build` work on `nanokvm-custom`
  and merge `upstream/NanoKVM` — **never** `upstream/main`; its own `CLAUDE.md`
  is the authority.
- On macOS the SDK checkout permanently shows ~38 files as modified. They are
  case-collision artifacts: never commit them, never `git add -A` there, and
  never build a kernel from a case-insensitive checkout.
- `third_party/` holds four submodules (`wstunnel`, `newt`, `usb-proxy`,
  `hev-socks5-tunnel`), each a fork on a branch named `NanoKVM` with an
  `upstream` remote already configured. `hev-socks5-tunnel` has nested
  submodules, so checkouts need `git submodule update --init --recursive`.
- To bump one: merge `upstream/main` on the `NanoKVM` branch, `git push origin
  NanoKVM`, then `git add third_party/<name>` and commit here, naming the
  upstream sha and every resolution in the message. Re-run `make tunnels` /
  `make passthrough` afterwards or `package.sh`'s `require_fresh` will refuse.
- The newt fork carries `f1ba26b` "strip unused subsystems" (removes the
  self-updater, Docker socket scanner, OTel, Prometheus, pprof). It must be
  re-applied on every upstream merge, and `NEWT_VERSION` in the `Makefile` must
  follow the newt tag that is an ancestor of the fork head.
- Builder-image gotcha: the local `nanokvm-builder-local-501-20` was built with
  uid 1000, so on a uid-501 host `make app` fails with `mkdir
  /home/build/.cache: permission denied`. Fix with `make rebuild-image`, or work
  around it with `make app DOCKER_TTY= UID=1000 GID=1000
  IMAGE_NAME=nanokvm-builder-local-501-20`.
- Host-side checks: `cd web && pnpm exec tsc --noEmit` and `cd server && gofmt -l .`
  work. Go builds and `go test ./...` on darwin fail by design (cgo against
  `dl_lib/libkvm.so`, plus Linux-only source) — use `make app` in the container.

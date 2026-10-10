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

Our patch renames the default import and unwraps `.default` when it is present, so
it is correct under either interop. **Delete it and restore the plain
`import Keyboard from 'react-simple-keyboard'` as soon as upstream resolves the
Vite 8 interop problem**, either by fixing the import themselves or by moving to a
Vite release that restores the behaviour.

## Workspace and builds

Read `docs/REPOSITORIES.md` before working outside this application. Merge
`upstream/main` here and `upstream/NanoKVM` in the companion SDK's
`nanokvm-custom` branch. Preserve each fork's explicit behavior and submodule
contracts. Inspect actual worktrees/remotes instead of assuming a machine layout.

Build the server with `make app DOCKER_TTY=` in the provided Docker builder; the
native application requires RISC-V cgo/libkvm. From `web`, `corepack pnpm test`
and `corepack pnpm build` run the frontend checks. Read `scripts/README.md` before
changing release packaging. Read the SDK's own `CLAUDE.md` before kernel work.

## Documentation contract

Documentation describes the current architecture, invariants, interfaces and
operating procedures. Keep it objective and independent of a particular session.
Do not add investigation diaries, dated implementation plans, chronological
reports, abandoned approaches or conversations. Git history records changes.
Update current guidance when behavior changes and remove obsolete instructions.

## Browser media and device operation

Browser webcam sharing lives in `web/src/pages/desktop/menu/devices/` and the
`sources`/`media` server packages. HDMI desktop streaming is a separate path in
`web/src/pages/desktop/screen/` and `server/service/stream/`. Read each package's
`CLAUDE.md` before changing it. An error in one path does not justify resetting
USB presentation, EDID, networking or the custom kernel.

Use installed Playwright and isolated headless browser contexts for UI checks;
do not attach to the operator's browser or reinstall existing browsers. Verify
real frame content and host display as well as transport counters. Mock assistant
requests and block remote HID sends when testing local assistant gestures.

Web-only deployment updates both `/kvmapp/server/web` and `/tmp/server/web`, with
assets first and `index.html` renamed last. Keep staging/backups under `/root`,
retain hashes needed by open tabs and verify the deployed assets. A hard refresh
loads the new code. See `docs/REPOSITORIES.md` for build and runtime details.

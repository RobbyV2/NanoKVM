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

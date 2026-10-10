# Web application

Use Node >=22 and pnpm >=11. `corepack pnpm` is available when the plain pnpm shim
is absent. Run `pnpm test`, `pnpm build`, and ESLint on changed files. The test
runner uses Node type stripping; worker URL imports require a test loader hook.

## Browser validation

Use installed Playwright and its existing browsers in isolated headless contexts.
Do not attach to the operator's session or reinstall browsers. Locate the installed
Node driver if it is provided by a Python Playwright package rather than a local
`node_modules/playwright`. Do not commit machine-specific package/cache paths.

Local built assets can be served at the device origin through `page.route` while
real APIs remain connected. Chromium then needs the test context's local-network
permission; grant `local-network-access`, `camera` and `microphone` together when
all are required, because a later grant replaces the earlier grant set. Validate
the deployed page without interception too.

The password notification's Ok button opens password settings. Its dismissal key
`nano-kvm-skip-modify-password` is JSON `{value: "true", expiry: <timestamp>}`.
The Devices action is **Share camera**. Test contexts may set their own dismissal
and toolbar preferences without changing the operator's local storage.

## Camera sharing

The attached host selects demand and geometry. A source lease alone does not
start a host camera. `/api/sources/events` is the ordered authority for sink state;
REST and source-handshake snapshots must not overwrite it. Status must reflect
black/silent fallback and unavailable state even when a local track is open.

Each camera start/restart owns a generation. Recheck it after permission, play and
bitmap awaits, and in worker callbacks. Retired attempts release late resources
without affecting successors; failures of the current attempt remain visible.
Provider ownership is per capture object, and Stop must prevent another capture
from starting while the release request is pending.

Use one test owner at a time. Verify real camera pixels, frame progression, source
output and the host's visible picture. A socket acknowledgment alone is not proof
of rendering. Exercise reconnect, pending startup cancellation and host reopen in
Chromium and Firefox; identify synthetic input explicitly.

## Assistant gestures

Ctrl-Ctrl shortcuts and quadruple click belong to the remote screen. Settings,
menus, popovers, local inputs and the assistant editor suppress them and clear
partial gesture state. Returning to the remote screen restores them. Tests must
mock `/api/assistant/*` and block HID output to avoid calling a real model or
sending test input to the host.

## Direct desktop decoding

Direct H.264 receives access units from the shared encoder. A client that misses
an access unit must recover at a keyframe before accepting dependent frames.
Linux Firefox below 157 uses software decoding because its VA-API canvas readback
can expose padded rows despite correct visibleRect metadata. The compatibility
policy is in `screen/decoder-compatibility.ts` and is explained in the UI; cropping
the picture does not repair decoder readback. Other browsers keep acceleration.

# nexit config file Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Make nexit usable on a Windows machine where the user cannot open cmd or PowerShell. They download `nexit.exe` and a `nexit.json` config file from the Exit panel, then double-click the exe. With no arguments, nexit finds the config by itself and connects.

**Architecture:** nexit gets a config loader that searches a fixed list of locations, first match wins. It is used whenever no address is given on the command line. The server serves a per-slot `nexit.json` next to the existing token-gated `nexit-windows-*.exe` assets. The Exit panel's nexit section gets download buttons for the binary (amd64/arm64) and the config. The existing cmd and PowerShell one-liners stay unchanged.

**Tech Stack:** Go (nexit module `nexit/`, server `server/service/exit`), React/antd (Exit settings panel).

**Spec:** the user's request (2026-09-24), quoted: "For nexit, the point is that one CANNOT open CMD or powershell. So, what Im thinking is, keep the script for nexit, still useful, but add it so that they can download the binary AND the config file, and run the binary, and it will auto pick up the config file in either the current directory or a user appdata directory OR a system directory (priority system -> user -> current directory)". Background: `docs/superpowers/specs/2026-09-15-exit-tunnel-design.md`, and the `nexit/1` protocol it defines.

## Global Constraints
- **File:** the config file is named `nexit.json`. It is JSON with the keys `address` (string, required), `passcode` (string, required), `insecure` (bool, optional, default false) and `allowPrivate` (bool, optional, default false). Unknown keys are an error (`DisallowUnknownFields`), so a typo can't pass silently.
- **Search order, first existing file wins, no merging:**
  1. System: `%ProgramData%\nexit\nexit.json`. On non-Windows, `/etc/nexit/nexit.json`.
  2. User: `os.UserConfigDir()` + `\nexit\nexit.json`, which is `%APPDATA%\nexit\nexit.json` on Windows.
  3. Current directory: `.\nexit.json`.
  4. Directory of the executable (`os.Executable()`): `nexit.json` beside the exe. This is an addition, placed after the three the user asked for. A shortcut or "open with" can start nexit with a working directory other than its folder.
- **When the config is used:** only when no address is given on the command line. With an address, behaviour is exactly as today. Flags given without an address (e.g. `--insecure`) override the matching config value. A passcode flag overrides the file's passcode.
- **Logging:** log which config file was used, as a path. Never log the passcode.
- **Launched with no arguments and config fails:** if nexit is started with no arguments and no config is found, or the config is invalid, it prints a clear message. The message lists every searched path and the expected JSON shape. Then it waits for Enter before exiting, so a double-clicked console window doesn't vanish. The wait happens only when stdin is a terminal/console. The same wait applies when a no-argument run ends with a fatal error.
- **Binary:** the binary stays identical for every device and slot (hash-stable, per the existing comment in `nexit/main.go`). Configuration still is never compiled in.
- **Serving the config:**
  - The server serves `<slot base>/nexit.json` behind the same bearer-token gate as `nexit-windows-*.exe`.
  - The body uses the exact values the existing nexit commands use: `address` = the base URL, `passcode` = the slot token, `insecure` = the same condition that adds `--insecure`.
  - `Content-Disposition: attachment; filename="nexit.json"`, `Cache-Control: no-store`.
- **Panel:**
  - The nexit section gets three buttons: "Download nexit (x64)", "Download nexit (ARM64)" and "Download nexit.json".
  - Each one fetches with the slot token as a Bearer header, the way the commands do, and saves the file as `nexit.exe` or `nexit.json` through a blob download.
  - Add a short instruction: put `nexit.json` next to `nexit.exe`, or in `%APPDATA%\nexit\`, or (admin) in `%ProgramData%\nexit\`, then double-click `nexit.exe`.
  - The one-liner commands stay.
- **i18n:** follow the Exit panel's existing locale convention. If its keys exist in all locales and a test enforces parity, add English text to every locale. Otherwise add them to `en.ts` only.
- **Security:** the config file holds the passcode, like the command line already does. Note in the panel text that the file contains the slot's passcode.
- **Checks:**
  - Go tests: `nexit` builds with `CGO_ENABLED=0`, so run `go test ./...` in `nexit/` via the builder container. `server/service/exit` is cgo-free and tested the same way.
  - Web: from `web/`, run `./node_modules/.bin/tsc --noEmit`, eslint, `node --experimental-strip-types --test "src/**/*.test.ts"` and `./node_modules/.bin/vite build`. pnpm is not on PATH.
  - Container: `docker run --rm -e UID=1000 -e GID=1000 -v "$PWD":/home/build/NanoKVM nanokvm-builder-local-1000-1000 /bin/bash -c 'cd /home/build/NanoKVM/<dir> && go test ./...'`. Run it from the worktree root.
- **Commits:** every commit message ends with `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`.

## Review Focus
1. A config file with a missing passcode or address, bad JSON or unknown keys gives an actionable message that names the file. It never gives a panic or a silent default.
2. A double-clicked exe whose config is missing keeps its window open with the list of searched paths.
3. Priority really is system → user → cwd → exe-dir. When several exist, the system file wins, and the log names it.
4. The token-gated `nexit.json` returns 401/404 without the bearer, exactly like the exe assets.
5. The passcode never appears in nexit's log output.

---

### Task 1: nexit config discovery (client)

**Files:**
- Create: `nexit/config.go`, `nexit/config_test.go`
- Modify: `nexit/main.go` (the no-address path, the usage text, the wait-for-Enter on a no-argument start)

**Interfaces:**
- `type fileConfig struct{ Address string \`json:"address"\`; Passcode string \`json:"passcode"\`; Insecure *bool \`json:"insecure"\`; AllowPrivate *bool \`json:"allowPrivate"\` }`
- `func configCandidates(env func(string) string, userConfigDir func() (string, error), cwd string, exeDir string, goos string) []string`. Pure: it returns the ordered candidate paths, so tests can pass a fake environment.
- `func loadConfig(paths []string, readFile func(string) ([]byte, error)) (cfg fileConfig, path string, err error)`. The first path whose file exists wins. A missing file is skipped. Any other read error is returned, naming the path. An error from any failure path must list every searched path when nothing was found.
- `main.go`:
  - If no address was given, get `configCandidates(os.Getenv, os.UserConfigDir, os.Getwd(), filepath.Dir(os.Executable()), runtime.GOOS)` and call `loadConfig`.
  - Apply the flags that were explicitly set (use `fs.Visit`) on top of the file values.
  - Validate that the address and passcode are non-empty.
  - Log `config: <path>`.
- If `len(os.Args) == 1` and the run fails, call `waitForEnter()`. It only waits when stdin is a character device (`os.Stdin.Stat()` mode `&os.ModeCharDevice`).

**Tests (table-driven):**
- candidate order on windows, with ProgramData/AppData set, and on linux
- an empty ProgramData is skipped
- the system file wins when all exist
- a missing file falls through to the next
- bad JSON, unknown key, missing passcode and missing address each give an error naming the path
- the not-found error lists all paths
- flag override (a unit test of the merge function, which should be split out as `mergeFlags(cfg fileConfig, set map[string]string/…)`; the implementer chooses the exact shape, but it must be unit-tested)

TDD. Commit: `nexit: find nexit.json in ProgramData, AppData, cwd or beside the exe`.

### Task 2: serve nexit.json and add download buttons to the Exit panel

**Files:**
- Modify: `server/service/exit/install.go` and/or `manager.go`, at the asset dispatch near `nexitAssets` (`manager.go` ~1443), to serve `nexit.json`
- Modify: `server/service/exit/commands.go`, only if the base/insecure computation must be shared. Extract a helper and do not duplicate it.
- Tests: the existing `server/service/exit/*_test.go` style. Cover: token gate, body fields, content type/disposition, and that `insecure` matches the command's `--insecure` condition.
- Modify: the Exit settings panel under `web/src/pages/desktop/menu/settings/exit/`, which renders `commands.nexit`, plus the i18n keys.
- If the panel needs data it doesn't have (e.g. the base URL for the asset fetch), extend `proto.GetExitCommandsRsp` in `server/proto/exit.go` and the TS type.

Commit: `exit: download nexit.exe and its nexit.json from the panel`.

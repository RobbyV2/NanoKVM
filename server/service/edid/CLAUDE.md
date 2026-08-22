# edid

Parses, validates, decodes and flashes the EDID the NanoKVM presents to the attached
host. The flash itself is performed by the shipped riscv64 C tool at
`/kvmapp/system/tool/nanokvm_update_edid`; this package drives it.
Design: `docs/superpowers/specs/2026-08-22-edid-design.md`.

## The child gets a plain pipe carrying "y\n"

`get_user_confirmation()` in `tools/nanokvm_update_edid/nanokvm_update_edid.c` runs from
`main` before the `argc` check and before the EDID file is opened, on alpha and beta
hardware. It is one `fgets` on stdin and it is the only stdin read in the program.

There is no `isatty`, no `tcgetattr`, no reopen of `/dev/tty` and no `TIOCGWINSZ`
anywhere in that file, so `fgets` on a pipe behaves exactly as `fgets` on a terminal for
that one read. A pty would add line-discipline echo, CRLF translation, SIGHUP-on-close
semantics and either cgo or a `creack/pty` dependency, to solve a problem the source says
does not exist. Do not add one.

Three ways to get the pipe wrong, all of which have been paid for once:

- Closed stdin or `/dev/null`. EOF is a decline, not consent. The tool prints
  `Input error. Exiting.` and exits 1 before touching the chip.
- A stream that stays open carrying anything whose first character is not `y` or `n`. The
  tool loops on `Invalid input. Please enter Y or N.` until EOF, so the wrapper hangs for
  the full 90 second timeout. Send exactly `"y\n"` and let the reader hit EOF.
- Omitting the file argument. The prompt fires before the `argc` check, so it consumes
  the `y` and then fails with the usage message.

On pcie the prompt is never reached and the extra `"y\n"` is harmless.

Reimplementing the flash in Go is rejected and should stay rejected. The three write
paths are sixty to ninety undocumented register pokes each, driving a SPI-flash erase and
program engine inside the LT6911 across three silicon variants with different chunk
sizes, different status registers and hand-tuned delays. A wrong step during an erase
does not produce an error, it produces a corrupt EDID region. Patching the C source is
also rejected: `scripts/package.sh` ships a committed prebuilt riscv64 musl ELF and the
Makefile wants a toolchain nothing in CI has.

## Outcomes come from stderr, never from the exit code

The tool has exactly two exit codes. `EDID data is invalid`, meaning the file was rejected
before a single I2C transaction and the chip is untouched, and
`EDID data mismatch after write/read cycle`, meaning the flash region is now in an unknown
state, are both exit 1. Success is exit 0 **and** `EDID data verified successfully` on
stdout, because the tool exits 0 on paths where the version page was never compared.

`stderrRows` in `apply.go` is the classification table and its order matters: the mismatch
row must win, because the tool follows that message with two 256-byte hex dumps that can
contain a word matching a later row.

## A mismatch is never retried

`needs_recovery` is the one outcome where the chip has been half written and there is no
way to find out how far. Retrying over a half-written flash region is how a recoverable
device becomes an unrecoverable one. `Apply` retries exactly one state, `bus_contention`,
where the tool failed to open or claim the bus and never reached the chip, and it retries
that once, re-taking the capture guard rather than reusing a stale one, then clears
`Retryable` so the API cannot loop either.

`needs_recovery` carries the two hex dumps back to the caller. On a device with no read
primitive those two blocks are the only diagnostic anyone will ever get.

## There is no read primitive, so rollback is re-flashing an archived file

`lt6911uxc_edid_read` and its two siblings are reachable only from inside
`lt6911_edid_config`, after a write has already completed. There is no dump mode, no
`--read`, and no way to snapshot what is on the chip before overwriting it. So
"transactional" here cannot mean capturing the pre-image, and `GET /api/vm/edid` reports
what NanoKVM last flashed and verified rather than what is on the chip. It says so in the
payload.

The archive is what makes rollback possible at all. `Store.Archive` runs only after stdout
carried the verification line, never before the spawn and never on a mismatch, because
the whole point of the archive is that it names bytes the chip accepted. The previous
`last-applied.bin` moves into `history/` first. Nothing is ever pruned from `history/`: a
newer write verifying says nothing about whether the operator wants the older EDID back,
and 256 bytes per apply is not a storage question. `E21_NanoKVM.bin` is the factory
restore target and the round-trip fixture.

On alpha and beta a completed flash does not take effect until the device is physically
power cycled, because the LT6911 reloads its EDID region only out of reset.
`RequiresPowerCycle` says so and nothing in the API claims the new EDID is live before
that.

## The whole flash is serialized against the capture pipeline

`kvm_vision.cpp` runs the HDMI detection thread against `/dev/i2c-4` and `hdmi.cpp`
constructs its I2C client at `0x2b`. That is the same bus and the same address the tool
drives. A detection read landing between an erase command and its status poll corrupts
the program sequence, and two interleaved program sequences corrupt the EDID region
outright.

So an apply takes, in order: the package-level `applyMu`, then the `O_CREATE|O_EXCL`
lockfile at `/etc/kvm/edid/.lock` carrying the pid, then the capture guard, held for the
whole child lifetime including the readback and released on every path including a
timeout kill. The lockfile exists because the mutex protects only this process; a stale
one is broken by checking `/proc/<pid>`. The guard is injected as `CaptureGuard` rather
than imported from `service/vm`, which would pull in the cgo capture library this package
tests without.

Preflight reads `/etc/kvm/hdmi_version` and `/etc/kvm/hw` here, with the tool's exact
`strcmp` semantics, and rejects the `ue` chip version before anything is spawned. It does
not use `config.GetHwVersion()`, which defaults unknown content to alpha where the C tool
fails, and would therefore prompt and flash on a board nobody has identified.

## The shipped library is 25 curated entries, not a corpus

`profiles_gen.go` is generated by `scripts/gen_edid_profiles.go` (`make edid-profiles`)
from 25 named upstream `linuxhw/EDID` paths fetched from pinned commit
`9c0c1bffc9c0f1cb2044115149a5ecb1652803f8`. Nothing is vendored and there is no submodule.
Generation is the only step that needs the network; a normal build reads the committed
table.

Curated rather than a corpus walk because of what this hardware can do. The capture path
tops out at 1920x1080 at 60 Hz, so an entry advertising 2560x1440 or 4K is an entry whose
preferred timing the KVM cannot display. A large table would be mostly modes this device
has no use for, and picking through it would be the operator's problem instead of the
generator's. The 25 are chosen for panel resolutions this device can actually display,
with 18 at 1920x1080p60 across different vendors so an operator can pick a plausible
monitor identity and not only a mode, plus `E21_NanoKVM.bin` as the factory entry.

Each entry carries its own `Source`, an upstream path plus the commit, so any shipped blob
traces back to one file at one revision. The generator drops a candidate that fails the
strict validator and logs the reason; at the current pin
`Digital/ASUS/AUS2426/5649967DE6D8` is the one drop. `library_test.go` re-checks the
committed table on every `go test`: both checksums recompute, the sha256 covers the blob,
provenance names a path and a 40-character commit or the factory blob, and decode followed
by encode reproduces the bytes exactly.

## The Go validator is deliberately stricter than the tool

`check_edid` in the C tool checks three things: length 256, the fixed header, and both
block checksums. A structurally nonsensical blob with correct checksums gets flashed, and
a 512-byte file is silently truncated to its first 256 bytes and flashed. `Decode` adds
version at least 1.3, extension count consistent with block 1, CTA structure, and at least
one DTD with a non-zero pixel clock, since a preferred timing of 0 Hz is a black screen on
the attached host. Keeping the validator strict is what keeps the classifier's
`invalid_input` row unreachable in practice.

The serializer recomputes both checksums rather than trusting the input, and preserves
unknown descriptor tags and unknown CTA data blocks byte for byte. `E21_NanoKVM.bin`
round-tripping to identical bytes is the single best test available here.

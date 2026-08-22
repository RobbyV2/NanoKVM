# EDID management

Status: design only. No code written, nothing has run on hardware.
Date: 2026-08-22.

## Goal

Let an administrator replace the EDID the NanoKVM presents to the attached host from the web UI, without a serial console and without hand-assembling a 256-byte blob. Program 2 covers parsing, validating, decoding, backing up and transactionally applying an EDID, a shipped profile library built from a vendored corpus, and the settings page that drives it.

Program 2 is independent of the USB presentation manager and of the Ethernet bridge. It shares no lock, no store and no code with either, and can be built in parallel with both. That is the last time either is mentioned.

`grep -rni edid server/ web/src/` returns only unrelated i18n substrings, so everything below is greenfield apart from `tools/nanokvm_update_edid/`, which ships a committed prebuilt riscv64 musl ELF and one binary EDID blob to `/kvmapp/system/tool/` via `scripts/package.sh:158-161`.

## Why the obvious wrapper fails

The obvious shape is `exec.Command("/kvmapp/system/tool/nanokvm_update_edid", path)`, wait, check the exit code. Every part of that is wrong.

The tool blocks on stdin before it validates its own arguments. `get_user_confirmation()` (`nanokvm_update_edid.c:775-804`) runs from `main` at `:878-889`, ahead of the `argc != 2` check and ahead of the `fopen` of the EDID file, on `PRODUCT_CUBE_A` and `PRODUCT_CUBE_B`. A child with an inherited or closed stdin does not fail fast, it either hangs or declines.

The exit code carries no information. There are exactly two values, 0 and 1, and every failure path returns `EXIT_FAILURE` or a bare `return 1`. `EDID data is invalid`, meaning the file you supplied was rejected before a single I2C transaction, and `EDID data mismatch after write/read cycle`, meaning the flash region is now in an unknown state and on Cube hardware needs a physical power cycle to even re-attempt, are the same integer.

Exit 0 does not mean the new EDID is live. On alpha and beta the tool prints its own instruction to power cycle physically rather than reboot (`:755-763`, `:769-773`), because the LT6911 reloads its EDID region only out of reset.

And there is no read primitive. `lt6911uxc_edid_read`, `lt6911c_edid_read` and `lt6911d_edid_read` are reachable only from inside `lt6911_edid_config`, after a write has already completed (`:692-716`). There is no dump mode, no `--read`, no way to snapshot what is on the chip before overwriting it, so "transactional" cannot mean "capture the pre-image".

## Decisions

| # | Decision |
|---|---|
| D1 | Drive the shipped C tool with a plain pipe carrying `"y\n"` on stdin. No pty, no patch to the tool, no reimplementation of the I2C sequences in Go. Argument in full below. |
| D2 | Classify outcomes by stderr text, never by exit code. `mismatch` is non-retryable and the API never auto-retries it. |
| D3 | Serialize against the capture pipeline. Hold `hdmiMutex` and `DisableHdmiCapture()` for the whole flash, behind an in-process mutex plus an on-disk lockfile. |
| D4 | Preflight `/etc/kvm/hdmi_version` and `/etc/kvm/hw` in Go with the tool's strict `strcmp` semantics, not `config.GetHwVersion()`. Reject the `ue` chip version before spawning anything. |
| D5 | Rollback is re-flashing an archived file, because there is no read primitive. Archive the flashed bytes only after stdout confirms verification, and treat the shipped `E21_NanoKVM.bin` as the factory restore target. A prior backup is never deleted because a newer write verified. |
| D6 | The Go validator is strictly stricter than `check_edid`. Nothing structurally nonsensical reaches the chip. |
| D7 | The profile library is a vendored linuxhw/EDID corpus validated at build time, with per-entry provenance, failures dropped and logged with a reason, compiled into a generated Go table. |
| D8 | The UI is a preset selector, a decoded summary of what is active, upload, two downloads and an apply action. The apply modal survives the general trim on Cube hardware, because recovering from a bad flash there needs someone to unplug the device. |

## What `nanokvm_update_edid` does today

Everything in this section is behaviour the wrapper depends on. Line numbers are `tools/nanokvm_update_edid/nanokvm_update_edid.c`.

### Command line

One mode, one argument, no flags, no getopt, no environment variables:

```
nanokvm_update_edid /path/to/edid.bin
```

`argc != 2` prints `Please enter the location of the EDID file using "<argv0> /path/to/edid.bin"` to stderr and exits 1 (`:886-889`). There is no `--force`, no `--yes`, no `--chip`, no `--device` override and no read mode.

### The prompt, and exactly when it blocks

```c
print_warning(product_version);
if (product_version == PRODUCT_CUBE_A || product_version == PRODUCT_CUBE_B) {
    if (get_user_confirmation() == 0) return EXIT_FAILURE;
}
// check command line arguments
if (argc != 2) { ... }
```

`get_user_confirmation()` is the only stdin read in the program, a bare `fgets(input, sizeof(input), stdin)` into a 256-byte buffer. It accepts any line whose first character lowercases to `y`, so `y`, `Y`, `yes` and `Yeah` all pass, and `n` declines. A blank line `continue`s without reprinting the prompt. Any other non-empty first character prints `Invalid input. Please enter Y or N.` and loops. EOF prints `\nInput error. Exiting.` and returns 0, which `main` turns into exit 1.

On `PRODUCT_PCIE_A` the function is never called and stdin is never read at all.

### Files it reads

`/etc/kvm/hdmi_version` (`VERSION_PATH`), read at `:818-829`, newline stripped at `:844`, compared with exact `strcmp` at `:847-862`:

| content | result |
|---|---|
| `c` | `CHIP_LT6911C`, stdout `Chip Version: LT6911C` |
| `ux` | `CHIP_LT6911UXC`, stdout `Chip Version: LT6911UXC` |
| `d` | `CHIP_LT6911D`, stdout `Chip Version: LT6911D` |
| `ue` | refused, stderr `Chip Version Error: UE version's edid can't be updated`, exit 1 |
| anything else | refused, stderr `Chip Version Error: Unknown version`, exit 1 |
| missing or unopenable | refused, stderr `Please upgrade to the latest system`, exit 1 |
| empty, `fgets` returns NULL | refused, stderr `Failed to read chip version`, exit 1 |

Because the match is `strcmp`, a CRLF or a trailing space yields `Unknown version`. The capture daemon is more lenient about the same file: `kvm_vision.cpp:1138-1172` reads two bytes, switches on `'u'` plus `'e'`/`'x'`, then `'d'`, and defaults to LT6911C, so the daemon and the tool can disagree about a malformed file. The file itself is produced by `/kvmapp/system/init.d/S15kvmhwd get_hdmi_version`.

`/etc/kvm/hw` (`PRODUCT_PATH`), read at `:831-842` and compared at `:864-876`:

| content | result |
|---|---|
| `alpha` | `PRODUCT_CUBE_A`, stdout `Product Version: CUBE_A` |
| `beta` | `PRODUCT_CUBE_B`, stdout `Product Version: CUBE_B` |
| `pcie` | `PRODUCT_PCIE_A`, stdout `Product Version : PCIE_A`, with the stray space before the colon |
| anything else | refused, stderr `Product Version Error: Unknown version`, exit 1 |
| missing or empty | `Please upgrade to the latest system` or `Failed to read product version`, exit 1 |

The EDID file, through `get_edid_from_file` (`:27-38`), which treats only `fopen` failure as an error, returns a short read as success with a small size, and silently truncates anything larger than 256 bytes to its first 256 bytes.

### Bus, address and chip variants

`I2C_DEVICE "/dev/i2c-4"` and `I2C_ADDRESS 0x2b`, opened at `:653-663` with `open(O_RDWR)` then `ioctl(client, I2C_SLAVE, 0x2b)`. All traffic afterwards is raw `read(2)` and `write(2)` on the fd, with no `I2C_RDWR` and no `i2c_smbus_*`, so every transaction is its own START and STOP. Register access is paged through `LT6911_REG_OFFSET 0xFF`, with a file-static `old_offset` cache that several call sites bypass by writing `0xFF` directly. Chunk sizes are 32 bytes for UXC and D, 16 for C.

Variant detection happens twice. The file-based layer above is the first, and `ue` is the refused variant. The second layer is a silicon ID inside each write path: UXC requires `i2c_read_byte(0x81, 0x08)` to be `0xEE` (`:285-289`), else stderr `Unsupported chip version`, and repeats the check post-write at `:327-330` with no message at all; C requires `i2c_read_bytes(0xA0, 0x00, buf, 2)` to be `16 05`, checked twice around a disable, 100 ms, enable cycle (`:379-395`); D has no silicon-ID check and instead reads a 32-byte version string out of flash first (`:490-526`), failing with `Failed to read LT6911D version data`. C also erase-verifies `i2c_read_byte(0x90, 0x02) == 0xFF`, else stderr `Clean Error` (`:419-423`), and repeats that check at `:448-449` without a message.

### Validation

`check_edid` (`:40-83`) checks exactly three things: length is exactly 256, bytes 0 through 7 are `00 FF FF FF FF FF FF 00`, and both block checksums hold, computed as `0x100 - sum` truncated to `uint8_t`, which is the standard `(-sum) & 0xFF`. It does not check the version bytes, the extension count, descriptor tags, DTD plausibility or the CTA tag. A structurally nonsensical blob with correct checksums is flashed. A 128-byte file is impossible, since byte 255 is always verified, and a zero-filled second block therefore requires byte 255 to be `0x00`.

### Readback and comparison

In `lt6911_edid_config` (`:689-740`): `sleep(1)`, a chip-specific read of 256 bytes into a fresh buffer, then `memcmp` against what was written. On match, stdout gets `EDID data verified successfully`. On mismatch, stderr gets `EDID data mismatch after write/read cycle` followed by both 256-byte buffers dumped as `%02X ` sixteen per line, and `main` prints `Failed to configure LT6911 EDID` and exits 1.

The verification is narrower than it looks. UXC and D write `256/32 + 1 = 9` blocks, where the ninth is a separate 32-byte version page at `0x5C` on page `0x81`. D populates that page from the string it read out of flash, and UXC writes `uint8_t version_str[32] = {0}` that is never populated (`:259`), so the UXC path unconditionally zeroes it. The readback loop covers `256/32 = 8` blocks, so that page is never compared, and `EDID data verified successfully` does not mean the whole write succeeded.

### GPIO reset

`NanoKVM_PCIe_HDMI_Reset()` (`:87-93`) drives `/sys/class/gpio/gpio451/value` low, sleeps 100 ms, drives it high, sleeps 100 ms, through two `system()` calls whose return values are ignored. It runs only for `PRODUCT_PCIE_A`, and twice, at `:903` before `lt6911_edid_config` and at `:911` after. It assumes gpio451 is already exported, and the export happens elsewhere, in `kvm_vision.cpp:2021-2023`, inside `kvmv_hdmi_control()` which early-returns unless `hw_version == 'p'`. A missing export makes the reset silently not happen.

### Exit codes

| exit | condition | line |
|---|---|---|
| 1 | `/etc/kvm/hdmi_version` unopenable, or empty | 820, 825 |
| 1 | `/etc/kvm/hw` unopenable, or empty | 833, 838 |
| 1 | chip version `ue`, or unknown | 857, 860 |
| 1 | product version unknown | 874 |
| 1 | user answered N, or stdin hit EOF | 881 |
| 1 | `argc != 2` | 888 |
| 1 | EDID file unopenable | 894 |
| 1 | EDID failed `check_edid` | 898 |
| 1 | write, silicon ID, erase, readback or compare failure | 908 |
| 0 | success | 915 |

Progress lines go to stdout, diagnostics go to stderr, so both streams are captured separately.

## Driving it

```go
ctx, cancel := context.WithTimeout(parent, 90*time.Second)
defer cancel()
cmd := exec.CommandContext(ctx, "/kvmapp/system/tool/nanokvm_update_edid", path)
cmd.Stdin = strings.NewReader("y\n")
var stdout, stderr bytes.Buffer
cmd.Stdout, cmd.Stderr = &stdout, &stderr
err := cmd.Run()
```

A pty buys nothing here. There is no `isatty`, no `tcgetattr` or other termios call, no reopen of `/dev/tty` and no `TIOCGWINSZ` anywhere in the file, so `fgets` on a pipe behaves exactly as `fgets` on a terminal for the one read that exists. A pty would add line-discipline echo, CRLF translation, SIGHUP-on-close semantics and either a cgo dependency or `creack/pty`, all to solve a problem the source says does not exist. Its one real side effect, unblocking musl's stdout buffering, matters only for live progress, which this design does not stream.

Three ways to get the pipe wrong:

- `< /dev/null`, or any closed stdin. EOF is a decline, not consent (`:781`), and the run exits 1 with `Input error. Exiting.` before touching the chip.
- Any stdin whose first character is neither `y` nor `n` on a stream that stays open. The tool loops on `Invalid input. Please enter Y or N.` until EOF (`:801`), so a wrapper feeding garbage hangs for the whole timeout. Send exactly `"y\n"` and let the reader hit EOF.
- Omitting the file argument. The prompt fires before the `argc` check, so it consumes the `y` and then fails with the usage message. Always pass the path.

On pcie the extra `"y\n"` is never read and is harmless.

Reimplementing the flash sequences in Go is rejected. The three write paths are sixty to ninety undocumented register pokes each, driving a SPI-flash erase and program engine embedded in the LT6911 across three silicon variants with different chunk sizes, different status registers and hand-tuned delays of 1 ms, 5 ms, 10 ms, 500 ms after erase and 600 ms after program. A wrong step during an erase produces no error, it produces a corrupt EDID region, and the server has no raw-I2C plumbing today, so this would be the tree's first `ioctl(I2C_SLAVE)` caller. Patching the C source is also rejected, because `scripts/package.sh:158-161` ships a committed prebuilt riscv64 ELF and the Makefile wants `riscv64-unknown-linux-musl-gcc` against a hardcoded sysroot at `/usr/local/RISC-V-toolchain/...`. A `--read out.bin` mode is the one patch that would ever justify standing that toolchain up in CI, and only if true pre-image rollback becomes a requirement.

## Classifying the result

Success is exit 0 **and** `EDID data verified successfully` present on stdout. Exit 0 without that line is recorded as unverified and does not archive.

| stderr contains | outcome | retryable | flash state |
|---|---|---|---|
| `EDID data mismatch after write/read cycle` | `needs_recovery` | never | unknown, region half written |
| `Unsupported chip version`, `Clean Error`, `Failed to read LT6911D version data` | `chip_refused` | after the operator acts | untouched, the chip refused before or during erase |
| `Failed to open the i2c bus`, `Failed to acquire bus access` | `bus_contention` | yes, once, after re-taking the capture guard | untouched |
| `Chip Version Error:`, `Product Version Error:`, `Please upgrade to the latest system`, `Failed to read chip version`, `Failed to read product version` | `preflight` | no | untouched, and unreachable if D4 did its job |
| `EDID data is invalid`, `EDID data length is not 256 bytes`, `EDID header is invalid`, `Checksum for ...` | `invalid_input` | no | untouched, and unreachable if D6 did its job |
| context deadline, no output | `timeout` | never | unknown |
| anything else with exit 1 | `generic` | no | unknown |

`needs_recovery` surfaces as a distinct state in the API and in the UI, carrying the two hex dumps the tool printed, offering the restore path, and refusing to retry. A retry loop over a half-written flash region is how a recoverable device becomes an unrecoverable one.

Success on alpha and beta carries `requires_power_cycle: true`. Nothing in the API claims the new EDID is live before that, and no verify endpoint can confirm it, because there is no read primitive to confirm with.

## Serializing against the capture pipeline

The capture daemon polls the same chip on the same bus. `kvm_vision.cpp:1252-1340` runs the HDMI detection thread against `/dev/i2c-4`, and `hdmi.cpp:6` constructs `i2c::I2C LT6911_i2c(4, i2c::Mode::MASTER)` at `LT6911_ADDR`, which is `0x2b`. A detection read landing between an erase command and its status poll corrupts the program sequence, and two interleaved program sequences from two concurrent flashers corrupt the EDID region outright.

So an apply takes, in order: the package-level `applyMu`, then an `O_CREATE|O_EXCL` lockfile at `/etc/kvm/edid/.lock` carrying the pid, then `hdmiMutex` and `DisableHdmiCapture()` from `server/service/vm/hdmi.go`, held for the whole child lifetime including the readback, and released with `EnableHdmiCapture()` in a deferred call that runs on every path including the timeout kill. The lockfile exists because the in-process mutex protects only this process, and a stale one is broken by checking `/proc/<pid>`.

On pcie the apply additionally confirms `/sys/class/gpio/gpio451/value` exists before spawning, since the tool's reset is a pair of ignored `system()` calls and a missing export makes it a no-op with no diagnostic.

## Preflight

Preflight reads `/etc/kvm/hdmi_version` and `/etc/kvm/hw` itself, strips exactly one trailing newline, and matches with the tool's `strcmp` table verbatim. It does not use `config.GetHwVersion()`. That helper (`server/config/hardware.go:57`) defaults unknown content to `HWVersionAlpha`, where the C tool fails, so using it would prompt and flash on a board nobody has identified. `ue` is rejected in Go, with a real message, before anything is spawned.

## The parser

`check_edid` is the floor, not the contract. The Go parser is strictly stricter, so that nothing structurally nonsensical reaches the flash and so that the classifier's `invalid_input` row stays unreachable.

### Hard rejects, which block the flash

1. Size exactly 256. Larger is rejected explicitly rather than truncated, because `get_edid_from_file` would accept the first 256 bytes of a 512-byte file and flash them.
2. Header bytes 0 through 7 equal `00 FF FF FF FF FF FF 00`.
3. `sum(0..127) % 256 == 0` and `sum(128..255) % 256 == 0`. A zero-filled extension requires byte 255 to be `0x00`, and a 128-byte file is impossible by construction.
4. EDID version and revision, bytes 18 and 19, at least 1.3.
5. Byte 126 is 1 with block 1 present, or byte 126 is 0 with block 1 all zero and a valid checksum. Any other combination is inconsistent and rejected.
6. If block 1 is a CTA block: byte 128 is `0x02`, byte 129 is at least 3, byte 130 (`d`) is in `[4, 127]`, and the DTD region `[d, 126)` parses cleanly.
7. At least one DTD with a non-zero pixel clock. A preferred timing of 0 Hz is a black screen on the attached host.

### Informational decode, which never blocks

- Manufacturer ID, 5-bit packed big-endian from bytes 8 and 9, so `4E 04` decodes to `SPD`. Product code little-endian from bytes 10 and 11, serial little-endian from 12 through 15, week and year from 16 and 17 with year equal to 1990 plus byte 17.
- Basic display parameters: byte 20 digital against analog, bytes 21 and 22 physical size in cm, byte 23 gamma as `v/100 + 1`, byte 24 feature bits covering DPMS states, colour type, sRGB default, preferred-timing-is-native and continuous frequency.
- Chromaticity, bytes 25 through 34, 10-bit packed pairs converted to CIE xy.
- Established timings I, II and manufacturer-reserved, bytes 35 through 37. Standard timings 1 through 8, bytes 38 through 53, including the `01 01` unused marker.
- All four 18-byte descriptors, bytes 54 through 125, dispatched on the `00 00 00 <tag>` prefix: `0xFF` display serial string, `0xFC` monitor name, `0xFD` range limits carrying vertical min and max, horizontal min and max, max pixel clock times 10 MHz and the secondary-formula byte, `0xF7` established timings III, `0x10` dummy, and everything else treated as a DTD.
- Full DTD decode: pixel clock as little-endian tens of kHz, the nibble-packed horizontal and vertical active, blanking, sync offset and sync width fields, image size in mm, borders, and flag byte 17 for interlace, stereo, sync type and horizontal and vertical sync polarity.
- CTA block: byte 131 for underscan, basic audio, YCbCr 4:4:4, YCbCr 4:2:2 and the native DTD count, then the data block collection walk over `[4, d)`, where each header byte gives tag `b>>5` and length `b&0x1F`, with tag 7 taking an extended tag byte. Tag 2 is the video data block, decoded to an SVD list with the bit-7 native flag stripped. Tag 3 is a vendor-specific block, decoded to its IEEE OUI, where `00-0C-03` is the HDMI 1.4 VSDB and `C4-5D-D8` is the HDMI Forum block, plus the source physical address. Tag 1 is audio and tag 4 is speaker allocation.
- Extension DTDs from `d` to 126.

### Serializer

The serializer exists for the profile library and for any future editor. It recomputes both checksums rather than trusting the input. It preserves unknown descriptor tags and unknown CTA data blocks byte for byte, so nothing that was not understood is dropped on the way back out. And it round-trips `E21_NanoKVM.bin` to identical bytes, which is the single best test available.

## `E21_NanoKVM.bin`

256 bytes, two blocks, both checksums valid. It is both the factory restore target and the round-trip fixture, so its decode is recorded here in full.

```
0000: 00 FF FF FF FF FF FF 00  4E 04 01 33 15 00 00 00
0010: 1E 23 01 03 80 30 1B 78  2A 69 25 A3 5B 50 A3 27
0020: 11 50 54 A5 4B 00 D1 C0  81 80 01 01 01 01 01 01
0030: 01 01 01 01 01 01 02 3A  80 18 71 38 2D 40 58 2C
0040: 45 00 DC 0C 11 00 00 1E  00 00 00 FF 00 4E 61 6E
0050: 6F 4B 56 4D 0A 20 20 20  20 20 00 00 00 FC 00 4E
0060: 61 6E 6F 4B 56 4D 0A 20  20 20 20 20 00 00 00 FD
0070: 00 38 4C 1E 53 11 00 0A  20 20 20 20 20 20 01 F5
0080: 02 03 11 30 46 04 1F 14  13 01 10 65 03 0C 00 10
0090: 00 02 3A 80 18 71 38 2D  40 58 2C 45 00 DC 0C 11
00A0: 00 00 1E 01 1D 00 72 51  D0 1E 20 6E 28 55 00 DC
00B0: 0C 11 00 00 1E 00 ... 00 (zeros through 0xFE)
00FF: DA
```

Base block:

| bytes | value | meaning |
|---|---|---|
| 0-7 | `00 FF FF FF FF FF FF 00` | fixed header |
| 8-9 | `4E 04` | manufacturer `SPD`, 5-bit packed, 19 S, 16 P, 4 D |
| 10-11 | `01 33` LE | product code `0x3301`, 13057 |
| 12-15 | `15 00 00 00` LE | serial 21 |
| 16 | `1E` | week 30 |
| 17 | `23` | year 2025, 1990 plus 35 |
| 18-19 | `01 03` | EDID 1.3 |
| 20 | `80` | digital input, no DFP 1.x flag |
| 21-22 | `30 1B` | 48 cm by 27 cm, 16:9 |
| 23 | `78` | gamma 2.20 |
| 24 | `2A` | active-off DPMS, colour type RGB 4:4:4 plus YCrCb 4:4:4, preferred timing is native, no continuous frequency |
| 25-34 | see below | chromaticity |
| 35 | `A5` | established I: 720x400@70, 640x480@60, 640x480@75, 800x600@60 |
| 36 | `4B` | established II: 800x600@75, 1024x768@60, 1024x768@75, 1280x1024@75 |
| 37 | `00` | manufacturer-reserved: none |
| 38-39 | `D1 C0` | standard timing 1, 1920x1080 @ 60, 16:9 |
| 40-41 | `81 80` | standard timing 2, 1280x1024 @ 60, 5:4 |
| 42-53 | `01 01` six times | standard timings 3 through 8 unused |
| 126 | `01` | one extension block |
| 127 | `F5` | base checksum |

Chromaticity, bytes 25 through 34 `69 25 A3 5B 50 A3 27 11 50 54`: red x 653 y 366 giving 0.6377 and 0.3574, green x 322 y 653 giving 0.3145 and 0.6377, blue x 156 y 70 giving 0.1523 and 0.0684, white x 321 y 337 giving 0.3135 and 0.3291. Approximately sRGB and Rec.709 primaries around a D65-ish white point.

Descriptor 1, bytes 54 through 71, `02 3A 80 18 71 38 2D 40 58 2C 45 00 DC 0C 11 00 00 1E`, is a DTD because bytes 0 and 1 are non-zero. Pixel clock `0x3A02` is 14850, so 148.50 MHz. H active 1920, H blank 280, H total 2200. V active 1080, V blank 45, V total 1125. H sync offset 88, H sync width 44, V sync offset 4, V sync width 5. Image size 476 by 268 mm, borders zero. Flag byte `0x1E` gives non-interlaced, no stereo, digital separate sync, positive VSync, positive HSync. That is 1920x1080p at exactly 60.000 Hz, the CEA-861 VIC 16 timing.

Descriptor 2, bytes 72 through 89, tag `0xFF` display serial number, payload `"NanoKVM"` plus `0x0A` plus five `0x20` pad bytes. Descriptor 3, bytes 90 through 107, tag `0xFC` monitor name, byte-identical payload. Descriptor 4, bytes 108 through 125, tag `0xFD` display range limits: offset flags `0x00`, vertical 56 to 76 Hz, horizontal 30 to 83 kHz, max pixel clock `0x11` times 10 giving 170 MHz, byte 10 `0x00` meaning default GTF with no secondary formula, then the mandated `0A` and six `20`. There is no `0xF7` established timings III and no `0x10` dummy, so all four descriptor slots carry real content.

Extension block, CTA-861: byte 128 `02` is the CTA tag, byte 129 `03` is revision 3, byte 130 `11` puts `d` at extension byte 17, byte 131 `30` gives underscan 0, basic audio 0, YCbCr 4:4:4 1, YCbCr 4:2:2 1, native DTD count 0.

The data block collection occupies extension bytes 4 through 16. The video data block header `0x46` is tag 2 length 6, payload `04 1F 14 13 01 10`, with no bit-7 native flags: VIC 4 is 1280x720p@60, VIC 31 is 1920x1080p@50, VIC 20 is 1920x1080i@50, VIC 19 is 1280x720p@50, VIC 1 is 640x480p@60, VIC 16 is 1920x1080p@60. The vendor specific block header `0x65` is tag 3 length 5, payload `03 0C 00 10 00`, IEEE OUI `00-0C-03` for HDMI Licensing LLC, source physical address 1.0.0.0, and nothing else: no max TMDS clock byte, no deep colour flags, no latency fields, no 3D or 4K VIC list, and no HF-VSDB, so no HDMI 2.0 advertisement. There is no audio data block, no speaker allocation block, no video capability block and no colorimetry block, which together with basic audio 0 means this EDID advertises no audio at all.

Extension DTD A, extension bytes 17 through 34, is byte-for-byte identical to base descriptor 1. Extension DTD B, extension bytes 35 through 52, `01 1D 00 72 51 D0 1E 20 6E 28 55 00 DC 0C 11 00 00 1E`, has pixel clock `0x1D01` giving 74.25 MHz, H active 1280, H blank 370, H total 1650, V active 720, V blank 30, V total 750, H sync offset 110, H sync width 40, V sync offset 5, V sync width 5, image size 476 by 268 mm, flags `0x1E`, which is 1280x720p at 60.000 Hz, the CEA VIC 4 timing. Extension bytes 53 through 126 are zero and the checksum at byte 255 is `0xDA`.

The round-trip test asserts 148500 kHz over 2200 over 1125 equals 60.000 Hz, and that parse followed by serialize reproduces all 256 bytes.

## Store, backup and rollback

```
/etc/kvm/edid/
  last-applied.bin       # the bytes of the last verified flash            0600
  last-applied.json      # sha256, decoded summary, source, timestamp      0600
  history/<ts>-<sha8>.bin  # every prior verified flash, never pruned      0600
  .lock                  # apply lockfile, pid inside                      0600
```

The directory is 0755 and every file is 0600, written through the `atomicFile` helper at `server/service/extensions/tunnel/config.go:85-152` verbatim, with a package-level `var edidDir` declared `var` rather than `const` so tests can swap it the way `tunnel/config_test.go:13-19` does.

`last-applied.bin` is written only after stdout carried `EDID data verified successfully`, never before the spawn and never on a mismatch, because the whole point of the archive is that it names bytes the chip accepted. The previous `last-applied.bin` moves into `history/` first. Nothing is ever deleted from `history/`, because a newer write verifying says nothing about whether the user wants the older EDID back, and 256 bytes per apply is not a storage question.

Rollback is re-running the tool against an archived file, which is the only rollback primitive that exists. The restore targets, in order of preference, are the most recent `history/` entry, then `/kvmapp/system/tool/E21_NanoKVM.bin` as the factory image. On a `needs_recovery` outcome the UI offers the factory image first and states that the device must be power cycled before the restore attempt will do anything on Cube hardware.

## Profile library

`third_party/linuxhw-edid` is vendored as a submodule pinned to a commit, and a build-time generator walks it. Every candidate is parsed and run through the full hard-reject validator, normalized to exactly 256 bytes, and either emitted or dropped with a reason logged. The generated Go table carries one row per survivor:

```go
type Profile struct {
    SHA256       [32]byte
    Manufacturer string // "DEL"
    Model        string // 0xFC monitor name, trimmed
    PreferredMode string // "1920x1080p60"
    Source       string // upstream path plus the pinned commit
    Data         []byte // exactly 256 bytes
}
```

`Source` is per entry rather than per corpus, so any shipped blob traces back to one upstream file at one commit. No blob ships whose second-block checksum the generator did not recompute itself. Generation runs from `make edid-profiles` and the generated file is committed, so a normal build needs neither the submodule nor the corpus.

The shipped set is small and hand-picked from the survivors rather than being the whole corpus: one entry per common resolution and refresh combination the KVM use case actually wants, plus `E21_NanoKVM.bin` as the factory entry.

## API

Routes join the existing admin group in `server/router/vm.go`, which already carries `CheckToken()` and `RequireRole(authn.RoleAdmin)`.

```
GET    /api/vm/edid            # active summary, decoded, plus preflight state
GET    /api/vm/edid/profiles   # the generated table, decoded summaries only
POST   /api/vm/edid/decode     # parse and decode an uploaded blob, no side effects
POST   /api/vm/edid/apply      # {profile} or {data}: validate, lock, flash, archive
POST   /api/vm/edid/restore    # {source: "factory"|"history", id}
GET    /api/vm/edid/download   # the active bytes
GET    /api/vm/edid/backup     # a named history entry
```

`GET /api/vm/edid` reports what NanoKVM last flashed and verified, not what is on the chip, and says so in the payload with an `unverified_since_boot` flag when no archive exists. There is no chip read to report.

`POST /apply` returns `{ "state": "...", "requires_power_cycle": bool, "message": "..." }` where `state` is one of the classifier outcomes above. `needs_recovery` additionally carries `written_hex` and `read_hex`, the two dumps the tool printed, because on a device with no read primitive those two blocks are the only diagnostic anybody will ever get.

`POST /decode` exists so the UI can show what an uploaded file means before anything is flashed, and it touches no lock, no chip and no store.

## UI

One settings tab, `web/src/pages/desktop/menu/settings/edid/`, following the tab registration in `settings/index.tsx` inside the existing `isAdmin` spread. It holds a preset selector over the profile table, a decoded summary of what is currently active (manufacturer, model, preferred mode, extension count, audio, sha256), an upload control that routes through `/decode` before it offers apply, download buttons for the active bytes and for a chosen backup, and one apply action.

The apply modal stays. Everywhere else in this program the rule is that modals are reserved for irreversible or self-locking actions, and this qualifies on both counts: on alpha and beta a completed flash does not take effect until someone physically disconnects power, and a mismatch leaves a region that cannot be diagnosed and cannot be re-attempted without that same physical access. The modal states the hardware variant, states whether a power cycle will be required, and requires the word to be confirmed rather than defaulting the primary button.

Conventions the build enforces: named arrow-function exports with no default export, `.ts` extensions on `@/` imports, an `if (isLoading) return` re-entrancy guard on every handler, `.then` and `.catch` and `.finally` rather than async, errors into a local `errMsg` rendered in `text-red-500`, and import ordering from `@ianvs/prettier-plugin-sort-imports`. `pnpm build` is `tsc && vite build` with `noUnusedLocals` and `noUnusedParameters`, so a dead import fails the build. All 24 locales get a real `settings.edid` block, including `settings.edid.title`, or the sidebar renders a raw key.

## Files

Backend:

| Path | Action |
|---|---|
| `server/proto/edid.go` | new: request and response types, the outcome enum |
| `server/service/edid/parse.go` | new: parser, hard rejects, informational decode |
| `server/service/edid/parse_test.go` | new: fixture round-trip, reject table |
| `server/service/edid/serialize.go` | new: checksum recompute, unknown-block preservation |
| `server/service/edid/tool.go` | new: spawn, stdin pipe, stream capture, stderr classifier |
| `server/service/edid/preflight.go` | new: strict `strcmp` reads of the two `/etc/kvm` files |
| `server/service/edid/apply.go` | new: lock, capture quiesce, spawn, classify, archive |
| `server/service/edid/store.go` | new: atomic store over `/etc/kvm/edid/`, history |
| `server/service/edid/profiles_gen.go` | new: generated, committed |
| `server/service/edid/service.go` | new: gin handlers |
| `server/router/vm.go` | modify: seven routes in the admin group |

Build and vendoring:

| Path | Action |
|---|---|
| `third_party/linuxhw-edid` | new submodule, pinned |
| `scripts/gen_edid_profiles.go` | new: corpus walk, validate, drop with reason, emit table |
| `Makefile` | modify: `edid-profiles` target |
| `.gitmodules` | modify |
| `tools/nanokvm_update_edid/` | unchanged, including the prebuilt binary |

Frontend:

| Path | Action |
|---|---|
| `web/src/api/vm/edid.ts` | new |
| `web/src/pages/desktop/menu/settings/edid/index.tsx` | new |
| `web/src/pages/desktop/menu/settings/edid/summary.tsx` | new: decoded view |
| `web/src/pages/desktop/menu/settings/edid/apply-modal.tsx` | new |
| `web/src/pages/desktop/menu/settings/index.tsx` | modify: one tab entry |
| `web/src/i18n/locales/*.ts` | modify: all 24 |

## Risks

**R2.1 A flash half succeeds, leaving the LT6911 EDID region in an unknown state with no read primitive to diagnose it.** `EDID data mismatch after write/read cycle` (`:720-735`) is the signature, there is no dump mode, and on Cube hardware recovery requires a physical power cycle. Retires when a deliberate-corruption test on real alpha and pcie hardware is run and documented: interrupt the child mid-write, then confirm that re-flashing `E21_NanoKVM.bin` after a power cycle recovers the device. Until that transcript exists, the API must not auto-retry a mismatch under any circumstances.

**R2.2 I2C contention with the capture daemon corrupts a flash program sequence.** `kvm_vision.cpp:1252-1340` and `hdmi.cpp:6` drive `/dev/i2c-4` at `0x2b` concurrently with the tool. Retires when an on-device run with `DisableHdmiCapture()` held shows zero `Failed to acquire bus access` and zero mismatch outcomes across 20 consecutive flashes, and the same test without the guard reproduces at least one failure, which is what proves the guard is doing the work rather than the timing being forgiving.

**R2.3 The linuxhw/EDID corpus contains blobs that pass both checksums but are structurally hostile**, with an extension count disagreeing with block 1, zero-pixel-clock DTDs, or unknown CTA tags, and one of them ships in the profile library. Retires when the build-time validator has been run over the entire imported corpus, every rejection is logged with its reason, the shipped table is generated only from survivors, and `E21_NanoKVM.bin` plus at least 200 corpus entries round-trip parse to serialize to identical bytes.

## Testing

The parser, the serializer and the classifier are pure and carry table-driven tests that run on an x86 development machine with no device.

`parse_test.go` round-trips `E21_NanoKVM.bin` from `tools/nanokvm_update_edid/` and asserts all 256 bytes back, then asserts the decoded fields against the table above: manufacturer `SPD`, product `0x3301`, serial 21, week 30, year 2025, version 1.3, one extension, preferred mode 1920x1080p60 at 148.5 MHz with 2200 by 1125 totals, secondary 1280x720p60, range limits 56 to 76 Hz and 30 to 83 kHz and 170 MHz, HDMI 1.4 VSDB with source physical address 1.0.0.0, and no audio. A separate case builds a blob with an unknown descriptor tag and an unknown CTA data block and asserts that serialization preserves both byte for byte.

The reject table covers each of the seven hard rejects with a minimal blob that fails exactly one of them, including a 257-byte file, which is the case the C tool would have truncated and flashed.

`tool_test.go` feeds the classifier captured stderr text for every row of the classification table and asserts the outcome and the retryable flag, with the mismatch case additionally asserting that the parsed hex dumps come back intact. Nothing in this repo mocks `exec.Command`, so the spawn itself is tested only by a fake binary written into `t.TempDir()` that reads stdin, echoes a canned stderr and exits with a chosen code, which covers the `"y\n"` handshake, the EOF-is-decline path and the timeout.

`store_test.go` asserts 0600 on every written file, mirroring `mcp/config_test.go:42-49`, asserts that a failed apply writes nothing, and asserts that a successful apply moves the previous `last-applied.bin` into `history/` rather than overwriting it.

Corpus validation runs from `make edid-profiles` and is the gate for R2.3: the whole vendored corpus through the validator, rejections logged with reasons, at least 200 survivors round-tripping to identical bytes.

No CI workflow in this repo runs `go test` or `go vet`, so these run locally and in review until that changes. `make web` does run `tsc && vite build`, so TypeScript errors do fail the build.

#!/bin/bash
#
# The EDID checks that go test cannot make on its own.
#
#   decode  Differential decode against edid-decode, the linuxtv reference
#           decoder, over every shipped profile and every testdata/corpus blob.
#           This is the "//go:build edidoracle" test, which needs edid-decode on
#           PATH and therefore does not run in a plain go test.
#
#   tool    Drives the shipped riscv64 nanokvm_update_edid under QEMU. Every
#           outcome the classifier in apply.go names is a string this binary
#           either does or does not print, and apply_test.go asserts against
#           strings a human typed. This runs the real ELF against real files and
#           checks the strings come back. Everything up to the first I2C
#           transaction is reachable; the chip is not, so the rows that need
#           silicon are listed as unreachable rather than faked.
#
#   device  The remaining half, on hardware. The tool reads the flash region
#           back and compares it itself, so a write that reports
#           "EDID data verified successfully" is a completed write-read-compare.
#           This flashes a known blob, checks that line, then flashes
#           /kvmapp/system/tool/E21_NanoKVM.bin back. It needs no monitor and no
#           human, and it does write to the chip.
#
# Usage: scripts/edid-repro.sh [decode|tool|all]
#        scripts/edid-repro.sh device <ssh target>

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_IMAGE="${GO_IMAGE:-golang:1.25}"
RV_IMAGE="${RV_IMAGE:-riscv64/alpine:edge}"
GOCACHE_VOLUME="${GOCACHE_VOLUME:-nanokvm-kernelint-gocache}"
TOOL="$ROOT/tools/nanokvm_update_edid"
CORPUS="$ROOT/server/service/edid/testdata/corpus"
DEVICE_BLOB="$CORPUS/Digital-Samsung-SAM0D22-F20C544C86F0.bin"

fail() {
    echo "edid-repro: $*" >&2
    exit 1
}

# The strings the tool prints are only useful while apply.go still matches on
# them, so every marker this script expects has to be in that table.
assert_classified() {
    grep -qF "\"$1\"" "$ROOT/server/service/edid/apply.go" ||
        fail "apply.go no longer classifies \"$1\""
}

run_decode() {
    docker run --rm -v "$ROOT:/w" -w /w/server -v "$GOCACHE_VOLUME:/root/.cache/go-build" "$GO_IMAGE" \
        sh -euc 'apt-get update -qq >/dev/null && apt-get install -y -qq edid-decode >/dev/null
                 edid-decode --version
                 go test -tags edidoracle -count=1 ./service/edid -run TestOracle -v' |
        grep -Ev '^(=== RUN|=== PAUSE|=== CONT)'
}

run_tool() {
    for marker in \
        "Please upgrade to the latest system" \
        "Failed to read chip version" \
        "Failed to read product version" \
        "Chip Version Error:" \
        "Product Version Error:" \
        "EDID data length is not" \
        "EDID header is invalid" \
        "Checksum for" \
        "EDID data is invalid" \
        "Failed to open the i2c bus" \
        "Failed to acquire bus access"; do
        assert_classified "$marker"
    done

    # The blobs the Go validator refuses for a structural reason are exactly the
    # ones check_edid has to accept, or "stricter than the tool" is not true.
    local stricter refused
    stricter="$(grep -oE '\{"[^"]+\.bin", Reject[A-Za-z]+\}' "$ROOT/server/service/edid/corpus_test.go" |
        grep -v RejectChecksum | sed -E 's/\{"([^"]+)".*/\1/' | tr '\n' ' ' || true)"
    refused="$(grep -oE '\{"[^"]+\.bin", RejectChecksum\}' "$ROOT/server/service/edid/corpus_test.go" |
        sed -E 's/\{"([^"]+)".*/\1/' | tr '\n' ' ' || true)"
    [ -n "$stricter" ] || fail "corpus_test.go lists no structurally rejected blob"

    docker run --rm -i --platform linux/riscv64 \
        -e STRICTER="$stricter" -e REFUSED="$refused" \
        -v "$TOOL:/tool:ro" -v "$CORPUS:/corpus:ro" \
        -v "$ROOT/server/service/edid/testdata/E21_NanoKVM.bin:/factory.bin:ro" \
        "$RV_IMAGE" /bin/sh -s <<'INNER'
set -u
cp /tool/nanokvm_update_edid /tmp/edid-tool
mkdir -p /etc/kvm /tmp/blobs
cp /factory.bin /tmp/blobs/factory.bin
cp /corpus/*.bin /tmp/blobs/ 2>/dev/null

head -c 255 /tmp/blobs/factory.bin > /tmp/short.bin
cp /tmp/blobs/factory.bin /tmp/badsum.bin
printf '\000' | dd of=/tmp/badsum.bin bs=1 seek=127 count=1 conv=notrunc 2>/dev/null
cp /tmp/blobs/factory.bin /tmp/badhdr.bin
printf '\000' | dd of=/tmp/badhdr.bin bs=1 seek=3 count=1 conv=notrunc 2>/dev/null

status=0
report() {
    if [ "$1" = "ok" ]; then
        echo "  ok   $2"
        return
    fi
    echo "  FAIL $2"
    status=1
}

# chip, product, blob, stdin, stream, expected string
case_is() {
    chip="$1"; product="$2"; blob="$3"; feed="$4"; stream="$5"; want="$6"
    rm -f /etc/kvm/hdmi_version /etc/kvm/hw
    # "-" leaves the file absent, "@" leaves it present and empty: fopen and
    # fgets fail on different lines and print different things.
    [ "$chip" = "-" ] || { [ "$chip" = "@" ] && : > /etc/kvm/hdmi_version; } || printf '%s\n' "$chip" > /etc/kvm/hdmi_version
    [ "$product" = "-" ] || { [ "$product" = "@" ] && : > /etc/kvm/hw; } || printf '%s\n' "$product" > /etc/kvm/hw

    if [ "$feed" = "eof" ]; then
        /tmp/edid-tool "$blob" > /tmp/out 2> /tmp/err < /dev/null
    else
        printf '%s\n' "$feed" | /tmp/edid-tool "$blob" > /tmp/out 2> /tmp/err
    fi
    code=$?

    if [ "$code" -eq 0 ]; then
        report bad "$want (exited 0, every failure path is 1)"
        return
    fi
    if grep -qF "$want" "/tmp/$stream"; then
        report ok "$stream carries \"$want\""
    else
        report bad "$want not in $stream: $(tr '\n' ' ' < /tmp/$stream | head -c 200)"
    fi
}

echo "preflight, before any I2C transaction"
case_is - -      /tmp/blobs/factory.bin y  err "Please upgrade to the latest system"
case_is @ alpha  /tmp/blobs/factory.bin y  err "Failed to read chip version"
case_is ux @     /tmp/blobs/factory.bin y  err "Failed to read product version"
case_is ue alpha /tmp/blobs/factory.bin y  err "Chip Version Error: UE version's edid can't be updated"
case_is zz alpha /tmp/blobs/factory.bin y  err "Chip Version Error: Unknown version"
case_is ux zz    /tmp/blobs/factory.bin y  err "Product Version Error: Unknown version"

echo "the confirmation pipe"
case_is ux alpha /tmp/blobs/factory.bin eof out "Input error. Exiting."
case_is ux alpha /tmp/blobs/factory.bin n   out "Do you want to continue?"

echo "check_edid, before any I2C transaction"
case_is ux pcie /tmp/short.bin  y err "EDID data length is not 256 bytes"
case_is ux pcie /tmp/short.bin  y err "EDID data is invalid"
case_is ux pcie /tmp/badsum.bin y err "Checksum for first 128 bytes is incorrect"
case_is ux pcie /tmp/badhdr.bin y err "EDID header is invalid"

echo "the bus, which is as far as this goes without the chip"
rm -f /dev/i2c-4
case_is ux pcie /tmp/blobs/factory.bin y err "Failed to open the i2c bus"
: > /dev/i2c-4
case_is ux pcie /tmp/blobs/factory.bin y err "Failed to acquire bus access and/or talk to slave"
rm -f /dev/i2c-4

echo "check_edid against every 256 byte blob in the repo"
printf 'ux\n' > /etc/kvm/hdmi_version
printf 'pcie\n' > /etc/kvm/hw
accepts() {
    printf 'y\n' | /tmp/edid-tool "$1" > /tmp/out 2> /tmp/err
    grep -q "EDID data loaded successfully" /tmp/out
}
for blob in /tmp/blobs/*.bin; do
    name="$(basename "$blob")"
    [ "$(wc -c < "$blob")" -eq 256 ] || continue
    case " $REFUSED " in
        *" $name "*)
            accepts "$blob" && report bad "$name reached the chip with a wrong checksum" ||
                report ok "$name refused by both validators"
            continue
            ;;
    esac
    case " $STRICTER " in
        *" $name "*)
            accepts "$blob" &&
                report ok "$name accepted by check_edid, refused by Decode" ||
                report bad "$name is not a case where Decode is the stricter one"
            continue
            ;;
    esac
    accepts "$blob" && report ok "$name passed check_edid" ||
        report bad "$name rejected: $(tr '\n' ' ' < /tmp/err | head -c 200)"
done

echo
echo "not reachable without the LT6911: EDID data mismatch after write/read cycle,"
echo "Unsupported chip version, Clean Error, Failed to read LT6911D version data,"
echo "EDID data verified successfully. Those need scripts/edid-repro.sh device."
exit "$status"
INNER
}

run_device() {
    local target="$1" chip product

    [ -f "$DEVICE_BLOB" ] || fail "$DEVICE_BLOB is missing"

    chip="$(ssh "$target" 'cat /etc/kvm/hdmi_version' | tr -d '\r\n')"
    product="$(ssh "$target" 'cat /etc/kvm/hw' | tr -d '\r\n')"
    echo "edid-repro: $target is chip $chip product $product"
    if [ "$chip" = "ue" ]; then
        fail "the ue chip cannot be flashed"
    fi

    ssh "$target" 'test -x /kvmapp/system/tool/nanokvm_update_edid && test -f /kvmapp/system/tool/E21_NanoKVM.bin' ||
        fail "the flash tool or the factory blob is missing on the device"

    echo "edid-repro: this writes the LT6911 flash region twice and restores the factory blob"
    scp -q "$DEVICE_BLOB" "$target:/tmp/edid-repro.bin"

    # The capture daemon polls the same chip at 0x2b on /dev/i2c-4, so nothing
    # else may be on the bus while the tool erases and programs.
    ssh "$target" '/etc/init.d/S95nanokvm stop >/dev/null 2>&1 || true'
    trap "ssh $target '/etc/init.d/S95nanokvm start >/dev/null 2>&1 || true'" EXIT

    device_flash "$target" /tmp/edid-repro.bin "the corpus blob"
    device_flash "$target" /kvmapp/system/tool/E21_NanoKVM.bin "the factory blob"
    ssh "$target" 'rm -f /tmp/edid-repro.bin'

    case "$product" in
        alpha|beta)
            echo "edid-repro: both writes verified; the chip reloads its EDID region only out of reset,"
            echo "edid-repro: so power cycle the device before believing what it presents"
            ;;
        *) echo "edid-repro: both writes verified" ;;
    esac
}

# Exit 0 alone is not success: the tool exits 0 on paths that never compared the
# readback.
device_flash() {
    local target="$1" blob="$2" what="$3" out

    out="$(ssh "$target" "printf 'y\n' | /kvmapp/system/tool/nanokvm_update_edid $blob 2>&1")" || true
    if ! printf '%s' "$out" | grep -q "EDID data verified successfully"; then
        printf '%s\n' "$out" >&2
        fail "flashing $what did not report a verified readback"
    fi
    echo "  ok   $what written and read back identical"
}

case "${1:-all}" in
    decode) run_decode ;;
    tool) run_tool ;;
    all)
        run_decode
        run_tool
        ;;
    device)
        [ $# -eq 2 ] || fail "usage: $0 device <ssh target>"
        run_device "$2"
        ;;
    *) fail "usage: $0 [decode|tool|all] | $0 device <ssh target>" ;;
esac

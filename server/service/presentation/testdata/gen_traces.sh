#!/bin/sh
# Regenerate the golden traces:
#   docker run --rm -v $PWD:/repo alpine:3.22 sh /repo/server/service/presentation/testdata/gen_traces.sh /repo
#
# Runs the unmodified logic of S03usbdev and S03usbhid under BusyBox ash against a
# sandbox tree. Only the four absolute roots the scripts reach outside the gadget
# (/etc/profile, /boot, /sys, /proc) are relocated into the sandbox by sed; every
# command, argument, escape sequence and branch is the script's own. mkdir, ln,
# cat, ls and echo are replaced by shell functions rather than PATH stubs because
# echo is an ash builtin and a PATH stub can never intercept it. Each function
# records the operation and then runs the genuine command, so the bytes in the
# trace are the bytes BusyBox echo -ne actually produced. The redirection target
# is recovered from /proc/self/fd, which is the only way a command can see where
# its stdout was pointed.

set -eu

repo=${1:-/repo}
init=$repo/kvmapp/system/init.d
out=$repo/server/service/presentation/testdata/traces
work=${TMPDIR:-/tmp}/nanokvm-golden

base_uid=nanokvm-golden-base-uid
udc=4340000.usb

rm -rf "$work"
mkdir -p "$work" "$out"

cat > "$work/prelude.sh" <<'PRELUDE'
_hex() { od -An -v -tx1 "$1" | tr -d ' \n'; }

_rel() {
    case $1 in
    "$G"/*) printf '%s' "${1#"$G"/}" ;;
    *) printf '%s' "$1" ;;
    esac
}

_trace_write() {
    case $1 in
    "$G"/*) printf 'write\t%s\t%s\n' "${1#"$G"/}" "$(_hex "$1")" >>"$TRACE" ;;
    "$SB/proc/cviusb/otg_role") printf 'otg\t%s\n' "$(_hex "$1")" >>"$TRACE" ;;
    esac
}

_passthrough() {
    exec 9>&1
    _target=$(readlink /proc/self/fd/9 2>/dev/null || :)
    exec 9>&-
    "$@"
    _status=$?
    _trace_write "$_target"
    return $_status
}

echo() { _passthrough command echo "$@"; }
cat() { _passthrough command cat "$@"; }
ls() { _passthrough command ls "$@"; }

_configfs_children() {
    case $1 in
    g0) command mkdir -p "$1/functions" "$1/configs" "$1/strings" "$1/os_desc" ;;
    configs/c.1) command mkdir -p "$1/strings" ;;
    functions/ncm.*) command mkdir -p "$1/os_desc/interface.ncm" ;;
    functions/rndis.*) command mkdir -p "$1/os_desc/interface.rndis" ;;
    functions/mass_storage.*) command mkdir -p "$1/lun.0" ;;
    esac
}

mkdir() {
    command mkdir "$@" || return $?
    printf 'mkdir\t%s\n' "$(_rel "$1")" >>"$TRACE"
    _configfs_children "$1"
}

ln() {
    command ln "$@" || return $?
    case $1 in -s) ;; *) return 0 ;; esac
    _dst=${3%/}
    printf 'symlink\t%s\t%s\n' "$_dst/${2##*/}" "$2" >>"$TRACE"
}
PRELUDE

# Phase B deltas. Everything recorded by the prelude above is the script's own
# behaviour; each delta below is one behaviour fix the compiler makes on top of
# it, applied here so the goldens stay generated rather than hand-edited.

# fix 1: normal mode writes the mode marker deliberately instead of leaning on
# the gadget core's kernel-version default, which a vendor bump would move (D6).
phase_b_bcddevice() {
    awk -F'\t' -v OFS='\t' -v marker="$(printf '0x0510\n' | od -An -v -tx1 | tr -d ' \n')" '
        !seen && $1 == "write" && $2 == "bcdDevice" { seen = 1 }
        !seen && $1 == "write" && $2 == "idVendor" { print "write", "bcdDevice", marker; seen = 1 }
        { print }
    ' "$1" >"$1.tmp"
    mv "$1.tmp" "$1"
}

# fix 2: the MS-OS block follows the functions actually in the plan, so a gadget
# with no network function clears os_desc/use and drops the os_desc/c.1 link
# instead of answering the 0xEE string request forever (H10, H11).
phase_b_os_desc() {
    awk -F'\t' -v OFS='\t' -v off="$(printf '0\n' | od -An -v -tx1 | tr -d ' \n')" '
        $1 == "mkdir" && $2 ~ /^functions\/(ncm|rndis)\./ { net = 1 }
        !seen && !net && $1 == "mkdir" && $2 ~ /^functions\// {
            print "write", "os_desc/use", off
            print "unlink", "os_desc/c.1"
            seen = 1
        }
        { print }
    ' "$1" >"$1.tmp"
    mv "$1.tmp" "$1"
}

# fix 3: the script's class=e0 is rejected by kstrtou8(page, 0, ...) and never
# reaches the IAD, so it goes; subclass=01 and protocol=03 are read as octal 1
# and 3 today and are written 0x-prefixed for the same values (H8).
phase_b_rndis_class() {
    awk -F'\t' -v OFS='\t' \
        -v subclass="$(printf '0x01\n' | od -An -v -tx1 | tr -d ' \n')" \
        -v protocol="$(printf '0x03\n' | od -An -v -tx1 | tr -d ' \n')" '
        $1 == "write" && $2 ~ /^functions\/rndis\.[^\/]+\/class$/ { next }
        $1 == "write" && $2 ~ /^functions\/rndis\.[^\/]+\/subclass$/ { print $1, $2, subclass; next }
        $1 == "write" && $2 ~ /^functions\/rndis\.[^\/]+\/protocol$/ { print $1, $2, protocol; next }
        { print }
    ' "$1" >"$1.tmp"
    mv "$1.tmp" "$1"
}

trace_case() {
    name=$1
    script=$2
    shift 2

    sb=$work/$name
    g=$sb/sys/kernel/config/usb_gadget/g0
    rm -rf "$sb"
    mkdir -p "$sb/boot" "$sb/etc" "$sb/proc/cviusb" "$sb/sys/class/cvi-base" \
        "$sb/sys/class/udc/$udc" "$sb/sys/kernel/config/usb_gadget"
    : >"$sb/etc/profile"
    : >"$sb/proc/cviusb/otg_role"
    printf '%s' "$base_uid" >"$sb/sys/class/cvi-base/base_uid"
    for flag in "$@"; do
        : >"$sb/boot/$flag"
    done

    sed -e "s#/etc/profile#$sb/etc/profile#g" \
        -e "s#/boot/#$sb/boot/#g" \
        -e "s#/sys/#$sb/sys/#g" \
        -e "s#/proc/#$sb/proc/#g" \
        "$init/$script" >"$sb/script.sh"

    trace=$out/$name.trace
    {
        printf '# generated by testdata/gen_traces.sh, do not edit\n'
        printf '# %s start\n' "$script"
        printf '# flags:%s\n' "$(for flag in "$@"; do printf ' %s' "$flag"; done)"
        printf '# plus the phase B deltas in gen_traces.sh\n'
    } >"$trace"

    (
        set +e
        SB=$sb G=$g TRACE=$trace
        export SB G TRACE
        cd "$sb/sys/kernel/config/usb_gadget"
        . "$work/prelude.sh"
        set -- start
        . "$sb/script.sh"
    ) >"$sb/stdout.log" 2>"$sb/stderr.log"

    phase_b_bcddevice "$trace"
    phase_b_os_desc "$trace"
    phase_b_rndis_class "$trace"

    duplicates=$(grep '^write' "$trace" | cut -f2 | sort | uniq -d)
    if [ -n "$duplicates" ]; then
        printf 'trace %s writes a path twice: %s\n' "$name" "$duplicates" >&2
        exit 1
    fi
    printf '%s: %s ops\n' "$name" "$(grep -vc '^#' "$trace")"
}

for mode in normal hidonly; do
    case $mode in
    normal) script=S03usbdev ;;
    hidonly) script=S03usbhid ;;
    esac
    for net in none ncm rndis ncmrndis; do
        case $net in
        none) netflags='' ;;
        ncm) netflags='usb.ncm' ;;
        rndis) netflags='usb.rndis0' ;;
        ncmrndis) netflags='usb.ncm usb.rndis0' ;;
        esac
        for disk in nodisk disk; do
            case $disk in
            nodisk) diskflags='' ;;
            disk) diskflags='usb.disk0' ;;
            esac
            trace_case "$mode.$net.$disk" "$script" $netflags $diskflags
        done
    done
done

trace_case normal.rndis.disk.bios S03usbdev usb.rndis0 usb.disk0 BIOS
trace_case normal.rndis.disk.notwakeup S03usbdev usb.rndis0 usb.disk0 usb.notwakeup
trace_case normal.rndis.disk.disablehid S03usbdev usb.rndis0 usb.disk0 disable_hid
trace_case normal.rndis.disk.diskro S03usbdev usb.rndis0 usb.disk0 usb.disk0.ro

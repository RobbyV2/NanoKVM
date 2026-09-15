#!/bin/sh
# Run S94exit under BusyBox ash, the shell the device has, against the same
# recording stubs the Go test uses on the host:
#
#   docker run --rm -v $PWD:/repo alpine:3.22 sh /repo/server/service/exit/testdata/ash_smoke.sh /repo
#
# s94exit_test.go covers the semantics under the host's sh; this exists because
# a bashism or a coreutils-only flag would pass there and fail on the device.
# It exits non-zero on the first difference from the Go test's expectations.

set -eu

repo=${1:-/repo}
script=$repo/kvmapp/system/init.d/S94exit
stubs=$repo/server/service/exit/testdata/stubs
work=${TMPDIR:-/tmp}/nanokvm-exit-ash

rm -rf "$work"
mkdir -p "$work/state/iptables" "$work/state/ip6tables" "$work/sys/class/net/usb0" \
	"$work/sys/kernel/config/usb_gadget/g0/configs/c.1/ncm.usb0" \
	"$work/etc/kvm/exit/0" "$work/var/run" "$work/bin" "$work/tmp" "$work/seed"
: > "$work/state/iptables/filter.FORWARD"
: > "$work/state/iptables/nat.PREROUTING"
: > "$work/state/iptables/nat.POSTROUTING"
: > "$work/state/ip6tables/filter.FORWARD"
echo usb0 > "$work/sys/kernel/config/usb_gadget/g0/configs/c.1/ncm.usb0/ifname"
echo 10.1.2.1/24 > "$work/state/addr.usb0"
printf '#!/bin/sh\nexit 0\n' > "$work/bin/hev-socks5-tunnel"
printf '#!/bin/sh\nexit 0\n' > "$work/bin/wstunnel"
chmod 755 "$work/bin/hev-socks5-tunnel" "$work/bin/wstunnel"
cat > "$work/etc/kvm/exit/0/env" <<'ENV'
SLOT=0
ENABLED=1
PENDING=0
MODE=wstunnel
ALLOW_PRIVATE=0
MTU=1280
NIC=usb0
REJECT4="0.0.0.0/8 127.0.0.0/8 169.254.0.0/16 198.18.0.0/15 224.0.0.0/3 10.0.0.0/8 100.64.0.0/10 172.16.0.0/12 192.168.0.0/16"
ENV

export PATH="$stubs:$PATH"
export STUB_STATE="$work/state" STUB_TRACE="$work/trace" STUB_ALIVE_PID=$$
export EXIT_DIR="$work/etc/kvm/exit" BIN_DIR="$work/bin" SEED_DIR="$work/seed"
export RUN_DIR="$work/var/run" LOG_DIR="$work/tmp" SYS_NET="$work/sys/class/net"
export GADGET_CONFIG="$work/sys/kernel/config/usb_gadget/g0/configs/c.1"

fail() {
	echo "ash smoke: $*" >&2
	echo "--- trace" >&2
	cat "$work/trace" >&2 2>/dev/null || true
	exit 1
}

echo "ash: $(sh --help 2>&1 | head -1)"

sh "$script" start 0 || fail "first start exited $?"
rules1=$(cat "$work/state/rules")
chain1=$(cat "$work/state/iptables/filter.EXIT0")
[ "$(grep -c . "$work/state/rules")" = 2 ] || fail "rules after first start: $rules1"
[ "$(grep -c . "$work/state/iptables/filter.EXIT0")" = 15 ] || fail "EXIT0 has $(grep -c . "$work/state/iptables/filter.EXIT0") rules"
grep -q "^-j EXIT0$" "$work/state/iptables/filter.FORWARD" || fail "no FORWARD jump"
grep -q "wstunnel-restrict.yml" "$work/trace" || fail "wstunnel not started"
[ -r "$work/var/run/exit0-hev.pid" ] || fail "no hev pidfile"

: > "$work/trace"
sh "$script" start 0 || fail "second start exited $?"
[ "$(cat "$work/state/rules")" = "$rules1" ] || fail "rules changed on the second converge"
[ "$(cat "$work/state/iptables/filter.EXIT0")" = "$chain1" ] || fail "chain changed on the second converge"
grep -q "start-stop-daemon -S" "$work/trace" && fail "second start restarted a live daemon"
grep -q "ip rule del" "$work/trace" && fail "second start deleted a live rule"

echo 1 > "$work/sys/class/net/exit0/carrier"
out=$(sh "$script" status 0) || fail "status exited non-zero with everything up: $out"
case "$out" in
	*"forward=1"*"routing=1"*"tun=1"*"hev=1"*"wstunnel=1"*"nat=1"*"nic=usb0"*"gw=10.1.2.1"*) ;;
	*) fail "status vector: $out" ;;
esac

sh "$script" stop 0 || fail "stop exited $?"
[ ! -s "$work/state/rules" ] || fail "rules survived stop"
[ ! -e "$work/state/iptables/filter.EXIT0" ] || fail "EXIT0 survived stop"
[ ! -s "$work/state/iptables/filter.FORWARD" ] || fail "FORWARD jump survived stop"
[ ! -e "$work/sys/class/net/exit0" ] || fail "tun survived stop"
[ ! -e "$work/var/run/exit0-hev.pid" ] || fail "hev pidfile survived stop"

echo "ash smoke: ok"

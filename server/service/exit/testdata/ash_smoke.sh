#!/bin/sh
# Run S94exit under BusyBox ash, the shell the device has, against the same
# recording stubs the Go test uses on the host:
#
#   docker run --rm -v $PWD:/repo alpine:3.22  sh /repo/server/service/exit/testdata/ash_smoke.sh /repo
#   docker run --rm -v $PWD:/repo busybox:1.37 sh /repo/server/service/exit/testdata/ash_smoke.sh /repo
#
# s94exit_test.go covers the semantics under the host's sh; this exists because
# a bashism or a coreutils-only flag would pass there and fail on the device.
# It exits non-zero on the first difference from the Go test's expectations.
#
# ip, iptables, ip6tables and sysctl are always the stubs. flock and
# start-stop-daemon are the real BusyBox applets when the image has both
# (busybox:1.37 does; alpine ships flock but not start-stop-daemon), and then
# the daemon section checks what only a real daemoniser can show: the slot lock
# is free once start returns, so the next converge does not block on it.

set -eu

repo=${1:-/repo}
script=$repo/kvmapp/system/init.d/S94exit
stubs=$repo/server/service/exit/testdata/stubs
work=${TMPDIR:-/tmp}/nanokvm-exit-ash

real=0
if command -v start-stop-daemon >/dev/null 2>&1 && command -v flock >/dev/null 2>&1
then
	real=1
fi

rm -rf "$work"
mkdir -p "$work/state/iptables" "$work/state/ip6tables" "$work/sys/class/net/usb0" \
	"$work/sys/kernel/config/usb_gadget/g0/configs/c.1/ncm.usb0" \
	"$work/etc/kvm/exit/0" "$work/var/run" "$work/bin" "$work/tmp" "$work/seed" "$work/stubs"
cp "$stubs"/* "$work/stubs/"
if [ "$real" = 1 ]
then
	rm -f "$work/stubs/flock" "$work/stubs/start-stop-daemon"
fi
: > "$work/state/iptables/filter.FORWARD"
: > "$work/state/iptables/nat.PREROUTING"
: > "$work/state/iptables/nat.POSTROUTING"
: > "$work/state/ip6tables/filter.FORWARD"
echo usb0 > "$work/sys/kernel/config/usb_gadget/g0/configs/c.1/ncm.usb0/ifname"
echo 10.1.2.1/24 > "$work/state/addr.usb0"
# The fake daemons announce themselves on stderr and then live until killed,
# like the real ones.
printf '#!/bin/sh\necho "fake hev-socks5-tunnel $*" >&2\nwhile :; do sleep 1; done\n' > "$work/bin/hev-socks5-tunnel"
printf '#!/bin/sh\necho "fake wstunnel $*" >&2\nwhile :; do sleep 1; done\n' > "$work/bin/wstunnel"
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

export PATH="$work/stubs:$PATH"
export STUB_STATE="$work/state" STUB_TRACE="$work/trace"
export EXIT_DIR="$work/etc/kvm/exit" BIN_DIR="$work/bin" SEED_DIR="$work/seed"
export RUN_DIR="$work/var/run" LOG_DIR="$work/tmp" SYS_NET="$work/sys/class/net"
export GADGET_CONFIG="$work/sys/kernel/config/usb_gadget/g0/configs/c.1"

cleanup() {
	for f in "$work"/var/run/*.pid "$work/state/daemons"
	do
		[ -r "$f" ] || continue
		while read -r pid
		do
			kill "$pid" 2>/dev/null || true
		done < "$f"
	done
}
trap cleanup EXIT

fail() {
	echo "ash smoke: $*" >&2
	echo "--- trace" >&2
	cat "$work/trace" >&2 2>/dev/null || true
	exit 1
}

# A converge that blocks on the slot lock must fail, not hang the run.
s94() {
	if command -v timeout >/dev/null 2>&1
	then
		timeout 20 sh "$script" "$@"
	else
		sh "$script" "$@"
	fi
}

echo "ash: $(sh --help 2>&1 | head -1); real flock/start-stop-daemon: $real"

s94 start 0 || fail "first start exited $?"
rules1=$(cat "$work/state/rules")
chain1=$(cat "$work/state/iptables/filter.EXIT0")
[ "$(grep -c . "$work/state/rules")" = 2 ] || fail "rules after first start: $rules1"
[ "$(grep -c . "$work/state/iptables/filter.EXIT0")" = 15 ] || fail "EXIT0 has $(grep -c . "$work/state/iptables/filter.EXIT0") rules"
grep -q "^-j EXIT0$" "$work/state/iptables/filter.FORWARD" || fail "no FORWARD jump"
[ -r "$work/var/run/exit0-hev.pid" ] || fail "no hev pidfile"
[ -r "$work/var/run/exit0-wstunnel.pid" ] || fail "no wstunnel pidfile"

if [ "$real" = 1 ]
then
	hevpid=$(cat "$work/var/run/exit0-hev.pid")
	wspid=$(cat "$work/var/run/exit0-wstunnel.pid")
	kill -0 "$hevpid" 2>/dev/null || fail "hev daemon $hevpid is not running"
	kill -0 "$wspid" 2>/dev/null || fail "wstunnel daemon $wspid is not running"
	# The lock lives on the open file description; a daemon that inherited
	# fd 9 holds it for its whole lifetime and every later converge blocks.
	( exec 9>"$work/var/run/exit0.lock"; flock -n -x 9 ) \
		|| fail "slot lock still held after start: fd 9 leaked into a daemon"
	for pid in $hevpid $wspid
	do
		ls -l "/proc/$pid/fd" 2>/dev/null | grep -q "exit0.lock" && fail "daemon $pid inherited the lock fd"
	done
	# -b gives the daemon /dev/null for stdio, so what wstunnel and hev write
	# on stderr reaches /tmp/exit0-*.log only through the wrapper that redirects
	# after the fork; without it the UI's log tail is always empty.
	sleep 1
	grep -q "fake wstunnel server --restrict-config" "$work/tmp/exit0-wstunnel.log" \
		|| fail "wstunnel stderr did not reach its log: $(cat "$work/tmp/exit0-wstunnel.log" 2>&1)"
	grep -q "fake hev-socks5-tunnel $work/etc/kvm/exit/0/hev.yml" "$work/tmp/exit0-hev.log" \
		|| fail "hev stderr did not reach its log: $(cat "$work/tmp/exit0-hev.log" 2>&1)"
fi

: > "$work/trace"
s94 start 0 || fail "second start exited $?"
[ "$(cat "$work/state/rules")" = "$rules1" ] || fail "rules changed on the second converge"
[ "$(cat "$work/state/iptables/filter.EXIT0")" = "$chain1" ] || fail "chain changed on the second converge"
grep -q "start-stop-daemon -S" "$work/trace" && fail "second start restarted a live daemon"
if [ "$real" = 1 ]
then
	[ "$(cat "$work/var/run/exit0-hev.pid")" = "$hevpid" ] || fail "second start restarted hev"
	[ "$(cat "$work/var/run/exit0-wstunnel.pid")" = "$wspid" ] || fail "second start restarted wstunnel"
fi
grep -q "ip rule del" "$work/trace" && fail "second start deleted a live rule"
grep -q "iptables -F EXIT0" "$work/trace" && fail "second start flushed EXIT0 outside a transaction"
grep -q "^iptables-restore -n$" "$work/trace" || fail "second start did not replace the chains through iptables-restore"

echo 1 > "$work/sys/class/net/exit0/carrier"
out=$(s94 status 0) || fail "status exited non-zero with everything up: $out"
case "$out" in
	*"forward=1"*"routing=1"*"tun=1"*"hev=1"*"wstunnel=1"*"nat=1"*"nic=usb0"*"gw=10.1.2.1"*) ;;
	*) fail "status vector: $out" ;;
esac

if [ "$real" = 1 ]
then
	# hev died and its pid was recycled by a process that is not hev: this
	# shell. status must not count it, and start must start a real hev.
	kill "$hevpid"; sleep 1
	echo $$ > "$work/var/run/exit0-hev.pid"
	out=$(s94 status 0) && fail "status exited 0 with a recycled hev pid: $out"
	case "$out" in
		*"hev=0"*) ;;
		*) fail "a recycled pid was reported as hev: $out" ;;
	esac
	s94 start 0 || fail "start with a recycled pid exited $?"
	hevpid=$(cat "$work/var/run/exit0-hev.pid")
	[ "$hevpid" != "$$" ] || fail "start trusted the recycled pid"
	kill -0 "$hevpid" 2>/dev/null || fail "restarted hev $hevpid is not running"
fi

s94 stop 0 || fail "stop exited $?"
[ ! -s "$work/state/rules" ] || fail "rules survived stop"
[ ! -e "$work/state/iptables/filter.EXIT0" ] || fail "EXIT0 survived stop"
[ ! -s "$work/state/iptables/filter.FORWARD" ] || fail "FORWARD jump survived stop"
[ ! -e "$work/sys/class/net/exit0" ] || fail "tun survived stop"
[ ! -e "$work/var/run/exit0-hev.pid" ] || fail "hev pidfile survived stop"
[ "$(cat "$work/state/sysctl/net.ipv4.ip_forward")" = 0 ] || fail "stop left ip_forward on"
[ "$(cat "$work/state/sysctl/net.ipv6.conf.usb0.disable_ipv6")" = 0 ] || fail "stop left IPv6 disabled on usb0"

if [ "$real" = 1 ]
then
	sleep 1
	kill -0 "$hevpid" 2>/dev/null && fail "stop left hev $hevpid running"
	kill -0 "$wspid" 2>/dev/null && fail "stop left wstunnel $wspid running"
fi

echo "ash smoke: ok"

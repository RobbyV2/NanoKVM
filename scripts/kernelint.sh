#!/bin/bash
#
# Runs the "//go:build linux && kernelint" tests, which mutate real kernel
# objects instead of a fake. Two tiers, because their kernel requirements are
# not the same:
#
#   tier1  service/bridge, service/passthrough
#          Needs a private network namespace, bridge, veth and vhci_hcd. Every
#          one of those is present on a GitHub ubuntu-latest runner, so this
#          tier runs there directly and costs seconds.
#
#   watchdog
#          A VM of its own with softdog and a stopless dw_wdt-shaped stub, to
#          settle what the watchdog core does when the process holding
#          /dev/watchdog dies. See run_watchdog.
#
#   tier2  service/presentation, service/functionfs, service/passthrough
#          Needs a UDC, which means dummy_hcd. No distro ships it
#          (CONFIG_USB_DUMMY_HCD is not set anywhere, and it is in neither
#          linux-modules nor linux-modules-extra), so the image built here
#          compiles it out of tree against linux-headers and boots a stock
#          Ubuntu kernel under QEMU. The -azure kernel on a GitHub runner has
#          CONFIG_USB_GADGET unset, so this tier cannot run on one directly.
#
# On a non-Linux host both tiers go through the VM: OrbStack's kernel is shared
# by every container and machine, and it carries no vhci_hcd and no gadget
# stack at all.
#
# Usage: scripts/kernelint.sh [tier1|tier2|watchdog|all]

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD="$ROOT/build/kernelint"

GO_IMAGE="${GO_IMAGE:-golang:1.25}"
VM_IMAGE="${VM_IMAGE:-nanokvm-kernelint-vm:4}"
WDT_IMAGE="${WDT_IMAGE:-nanokvm-watchdog-vm:1}"
KVER="${KVER:-6.8.0-138-generic}"
KTAG="${KTAG:-v6.8}"
ARCH="${ARCH:-arm64}"

TIER1_PACKAGES="bridge passthrough"
TIER2_PACKAGES="presentation functionfs passthrough"

# A tier that matched no test is exactly the failure mode this suite exists to
# catch, so a run counts what passed rather than trusting an "ok" line.
assert_ran() {
    local log="$1" least="$2" passed
    passed="$(grep -c -- '--- PASS' "$log" || true)"
    if [ "$passed" -lt "$least" ]; then
        echo "kernelint: $passed tests passed, expected at least $least" >&2
        exit 1
    fi
    echo "kernelint: $passed tests passed"
}

host_arch() {
    if command -v go >/dev/null 2>&1; then
        go env GOARCH
    else
        docker version --format '{{.Server.Arch}}'
    fi
}

# Go is not native on every host this runs on, so a missing toolchain falls back
# to the container rather than to an error.
compile() {
    local tier="$1" goarch="$2"
    shift 2
    mkdir -p "$BUILD/$tier"
    local script=""
    for package in "$@"; do
        script="$script go test -c -tags kernelint -o /w/build/kernelint/$tier/$package.test ./service/$package;"
    done
    if command -v go >/dev/null 2>&1; then
        for package in "$@"; do
            ( cd "$ROOT/server" && CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" \
                go test -c -tags kernelint -o "$BUILD/$tier/$package.test" "./service/$package" )
        done
        return
    fi
    docker run --rm \
        -v "$ROOT:/w" -w /w/server \
        -v nanokvm-kernelint-gocache:/root/.cache/go-build \
        -e CGO_ENABLED=0 -e GOOS=linux -e "GOARCH=$goarch" \
        "$GO_IMAGE" sh -euc "$script"
}

# The namespace is what makes this safe on a developer's machine and on a shared
# runner: every link the tests add and delete is inside it.
write_tier1_payload() {
    cat > "$BUILD/tier1/run.sh" <<'EOF'
#!/bin/sh
set -u
dir="${KERNELINT_DIR:-/tier1}"
mkdir -p /run /var/run
for module in vhci-hcd bridge veth; do
    modprobe "$module" 2>/dev/null
done
ip netns add kernelint
status=0
for binary in "$dir"/*.test; do
    ip netns exec kernelint "$binary" -test.v -test.run '^TestKernelTier1' || status=1
done
ip netns del kernelint
exit $status
EOF
    chmod +x "$BUILD/tier1/run.sh"
}

# One boot per package. configfs is global to the VM and f_hid attributes are
# -EBUSY while the function is linked, so a gadget another package left behind is
# not a gadget this one can write to.
write_tier2_payload() {
    cat > "$BUILD/tier2/run.sh" <<EOF
#!/bin/sh
set -u
for module in libcomposite dummy_hcd usb_f_fs usb_f_hid usb_f_ncm usb_f_uvc usb_f_uac2 usbhid hid-generic; do
    modprobe "\$module" || echo "kernelint: modprobe \$module failed"
done
mkdir -p /sys/kernel/config
mount -t configfs none /sys/kernel/config
mkdir -p /etc/kvm/presentation /etc/kvm/passthrough
/tier2/$1.test -test.v -test.run '^TestKernelTier2'
EOF
    chmod +x "$BUILD/tier2/run.sh"
}

vm_image() {
    docker image inspect "$VM_IMAGE" >/dev/null 2>&1 && return 0

    local context="$BUILD/img"
    mkdir -p "$context"

    cat > "$context/runvm.sh" <<'EOF'
#!/bin/bash
set -e
work=$(mktemp -d)
mkdir -p "$work/x"
cp -a /script/. "$work/x/"
chmod +x "$work/x/run.sh"
( cd "$work/x" && find . | cpio -o -H newc --quiet | gzip -1 ) > "$work/extra.cpio.gz"
cat /vm/initrd.img "$work/extra.cpio.gz" > "$work/combined.img"
exec qemu-system-aarch64 \
  -M virt -cpu max -smp "${VMCPUS:-4}" -m "${VMMEM:-4096}" \
  -kernel /vm/vmlinuz -initrd "$work/combined.img" \
  -append "console=ttyAMA0 panic=1 loglevel=4" \
  -nographic -no-reboot
EOF

    cat > "$context/Dockerfile" <<'EOF'
FROM ubuntu:24.04 AS build
ARG KVER
ARG KTAG
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
      linux-image-${KVER} linux-modules-${KVER} linux-modules-extra-${KVER} \
      linux-headers-${KVER} build-essential kmod iproute2 iputils-ping \
      ca-certificates curl cpio zstd \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /src/dummy
# dummy_hcd comments out its three isochronous EP_INFO pairs, so f_uvc and
# f_uac2 find no ISO endpoint to autoconfigure and never bind. Uncommenting
# them lets the rig verify enumeration and descriptors, and nothing else:
# PIPE_ISOCHRONOUS still returns -EINVAL because dummy_hcd never implemented
# ISO scheduling, so no conclusion about throughput or timing can be drawn
# here. That is a limit of the emulator, not of NanoKVM.
RUN curl -fsSL -o dummy_hcd.c "https://git.kernel.org/pub/scm/linux/kernel/git/stable/linux.git/plain/drivers/usb/gadget/udc/dummy_hcd.c?h=${KTAG}" \
 && perl -0777 -i -pe 'BEGIN{$n=0} $n += s{^/\*\n((?:\tEP_INFO\("ep\d+(?:in|out)-iso",\n\t\tUSB_EP_CAPS\(USB_EP_CAPS_TYPE_ISO, USB_EP_CAPS_DIR_(?:IN|OUT)\)\),\n){2})\*/\n}{$1}gm; END{die "uncommented $n iso endpoint pairs, expected 3\n" unless $n == 3}' dummy_hcd.c \
 && printf 'obj-m += dummy_hcd.o\n' > Makefile \
 && make -C /lib/modules/${KVER}/build M=/src/dummy modules \
 && install -m644 dummy_hcd.ko /lib/modules/${KVER}/kernel/drivers/usb/gadget/udc/dummy_hcd.ko

# f_uvc pulls in the V4L2 core and f_uac2 the ALSA core, neither of which is
# under kernel/drivers/usb; copying either subsystem whole would multiply the
# initrd, so only what modprobe says they need comes across.
RUN mkdir -p /rootfs/lib/modules/${KVER} \
 && cd /lib/modules/${KVER} \
 && cp modules.builtin modules.builtin.modinfo modules.order /rootfs/lib/modules/${KVER}/ \
 && for d in kernel/drivers/usb kernel/drivers/hid kernel/net/bridge kernel/net/llc kernel/net/802; do \
      tar cf - "$d" | (cd /rootfs/lib/modules/${KVER} && tar xf -); \
    done \
 && for module in usb_f_uvc usb_f_uac2; do modprobe -S ${KVER} -n --show-depends "$module" | awk '{print $2}'; done \
      | sed "s|^/lib/modules/${KVER}/||" | sort -u > /tmp/media-modules \
 && test -s /tmp/media-modules \
 && tar cf - -T /tmp/media-modules | (cd /rootfs/lib/modules/${KVER} && tar xf -) \
 && m=$(find kernel/drivers/net -maxdepth 1 -name 'veth.ko*' | head -1) \
 && tar cf - "$m" | (cd /rootfs/lib/modules/${KVER} && tar xf -) \
 && depmod -b /rootfs ${KVER}

RUN set -e; cd /rootfs \
 && mkdir -p bin sbin usr/bin usr/sbin etc proc sys dev tmp run lib/aarch64-linux-gnu usr/lib/aarch64-linux-gnu \
 && cp -a /bin/. bin/ && cp -a /sbin/. sbin/ && cp -a /usr/bin/. usr/bin/ && cp -a /usr/sbin/. usr/sbin/ \
 && cp -a /lib/aarch64-linux-gnu/. lib/aarch64-linux-gnu/ \
 && cp -a /usr/lib/aarch64-linux-gnu/. usr/lib/aarch64-linux-gnu/ \
 && cp -a /lib/ld-linux-aarch64.so.1 lib/ \
 && cp -a /etc/passwd /etc/group /etc/nsswitch.conf /etc/hosts etc/ \
 && rm -rf usr/share/doc usr/share/man usr/share/locale \
 && printf '%s\n' '#!/bin/sh' \
      'mount -t proc none /proc' \
      'mount -t sysfs none /sys' \
      'mount -t devtmpfs none /dev 2>/dev/null' \
      'mkdir -p /dev/pts && mount -t devpts none /dev/pts 2>/dev/null' \
      'export PATH=/usr/sbin:/usr/bin:/sbin:/bin' \
      '/run.sh; echo "RUNSH_EXIT=$?"' \
      'sync; echo 1 > /proc/sys/kernel/sysrq; echo o > /proc/sysrq-trigger; sleep 5' > init \
 && chmod +x init \
 && find . | cpio -o -H newc --quiet | gzip -1 > /initrd.img \
 && cp /boot/vmlinuz-${KVER} /vmlinuz

FROM alpine:3.22
RUN apk add --no-cache qemu-system-aarch64 cpio bash
COPY --from=build /initrd.img /vm/initrd.img
COPY --from=build /vmlinuz /vm/vmlinuz
COPY runvm.sh /usr/local/bin/runvm
RUN chmod +x /usr/local/bin/runvm
ENTRYPOINT ["/usr/local/bin/runvm"]
EOF

    docker build --platform "linux/$ARCH" \
        --build-arg "KVER=$KVER" --build-arg "KTAG=$KTAG" \
        -t "$VM_IMAGE" "$context"
}

# The payload lands at / inside the VM as a second cpio appended to the
# initramfs, so there is no shared filesystem and no state between runs.
vm_run() {
    local tier="$1"
    local log="$BUILD/$tier.log"
    local before
    before=$(wc -l < "$log" 2>/dev/null || echo 0)
    vm_image
    docker run --rm --platform "linux/$ARCH" \
        -v "$BUILD/$tier:/script/$tier:ro" -v "$BUILD/$tier/run.sh:/script/run.sh:ro" \
        "$VM_IMAGE" 2>&1 | tee -a "$log"
    tail -n +$((before + 1)) "$log" | grep -q '^RUNSH_EXIT=0' || {
        echo "kernelint: $tier payload exited non-zero" >&2
        exit 1
    }
}

# The watchdog probe. The trial guard in S02abtrial exists because
# /dev/watchdog was believed unable to notice the server dying: dw_wdt sets
# WDIOF_MAGICCLOSE, and the claim was that a close without a "V" leaves the
# kernel petting the device forever. That was read from source and never run,
# so it is run here. softdog covers the watchdog core; stopless_wdt is a
# dw_wdt-shaped stub whose stop() cannot stop (dw_wdt with no reset control
# only sets WDOG_HW_RUNNING) and which reports a max_hw_heartbeat_ms, because
# that pair is what decides whether the core's own keepalive worker takes over.
write_watchdog_payload() {
    cat > "$BUILD/watchdog/run.sh" <<EOF
#!/bin/sh
set -u
driver=$1
probe=$2
EOF
    cat >> "$BUILD/watchdog/run.sh" <<'EOF'
mkdir -p /run /var/run
if [ "$driver" = softdog ]; then
    modprobe softdog soft_margin=5 nowayout=0
else
    modprobe stopless_wdt nowayout=0
fi
dev=/dev/watchdog0
[ -e "$dev" ] || dev=/dev/watchdog
echo "PROBE $driver $probe START"
# The magic character has to come from the process that holds the fd: the
# watchdog core allows one opener, so a second open only returns -EBUSY.
if [ "$probe" = close-magic ]; then
    { exec 3<>"$dev"; printf 1 >&3; printf V >&3; exec sleep 999; } &
else
    { exec 3<>"$dev"; printf 1 >&3; exec sleep 999; } &
fi
holder=$!
sleep 2
if [ "$probe" = freeze ]; then
    kill -STOP "$holder"
else
    kill -9 "$holder"
fi
sleep 25
echo "PROBE $driver $probe SURVIVED"
dmesg | tail -n 20
EOF
    chmod +x "$BUILD/watchdog/run.sh"
}

wdt_image() {
    docker image inspect "$WDT_IMAGE" >/dev/null 2>&1 && return 0

    local context="$BUILD/wdtimg"
    mkdir -p "$context"
    cat > "$context/runvm.sh" <<'EOF'
#!/bin/bash
set -e
work=$(mktemp -d)
mkdir -p "$work/x"
cp -a /script/. "$work/x/"
chmod +x "$work/x/run.sh"
( cd "$work/x" && find . | cpio -o -H newc --quiet | gzip -1 ) > "$work/extra.cpio.gz"
cat /vm/initrd.img "$work/extra.cpio.gz" > "$work/combined.img"
exec qemu-system-aarch64 \
  -M virt -cpu max -smp "${VMCPUS:-2}" -m "${VMMEM:-4096}" \
  -kernel /vm/vmlinuz -initrd "$work/combined.img" \
  -append "console=ttyAMA0 panic=1 loglevel=7" \
  -nographic -no-reboot
EOF

    cat > "$context/stopless_wdt.c" <<'EOF'
#include <linux/module.h>
#include <linux/reboot.h>
#include <linux/timer.h>
#include <linux/watchdog.h>

static bool nowayout;
module_param(nowayout, bool, 0444);

static struct watchdog_device wdd;
static struct timer_list ticker;

static void stopless_fire(struct timer_list *unused)
{
	pr_emerg("stopless_wdt: expired, resetting\n");
	emergency_restart();
}

static int stopless_start(struct watchdog_device *w)
{
	mod_timer(&ticker, jiffies + w->timeout * HZ);
	return 0;
}

static int stopless_ping(struct watchdog_device *w)
{
	mod_timer(&ticker, jiffies + w->timeout * HZ);
	return 0;
}

/* dw_wdt_stop() with no reset control: it cannot stop, so it tells the core
 * the hardware is still running and returns success. */
static int stopless_stop(struct watchdog_device *w)
{
	pr_info("stopless_wdt: stop() cannot stop, hardware keeps running\n");
	set_bit(WDOG_HW_RUNNING, &w->status);
	return 0;
}

static const struct watchdog_info stopless_info = {
	.options = WDIOF_SETTIMEOUT | WDIOF_KEEPALIVEPING | WDIOF_MAGICCLOSE,
	.identity = "stopless_wdt",
};

static const struct watchdog_ops stopless_ops = {
	.owner = THIS_MODULE,
	.start = stopless_start,
	.stop = stopless_stop,
	.ping = stopless_ping,
};

static int __init stopless_init(void)
{
	timer_setup(&ticker, stopless_fire, 0);
	wdd.info = &stopless_info;
	wdd.ops = &stopless_ops;
	wdd.min_timeout = 1;
	wdd.max_timeout = 60;
	wdd.timeout = 5;
	wdd.max_hw_heartbeat_ms = 10000;
	watchdog_set_nowayout(&wdd, nowayout);
	return watchdog_register_device(&wdd);
}

static void __exit stopless_exit(void)
{
	watchdog_unregister_device(&wdd);
	timer_shutdown_sync(&ticker);
}

module_init(stopless_init);
module_exit(stopless_exit);
MODULE_LICENSE("GPL");
EOF

    cat > "$context/Dockerfile" <<'EOF'
FROM ubuntu:24.04 AS build
ARG KVER
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
      linux-image-${KVER} linux-modules-${KVER} linux-headers-${KVER} \
      build-essential kmod cpio \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /src/wdt
COPY stopless_wdt.c .
RUN printf 'obj-m += stopless_wdt.o\n' > Makefile \
 && make -C /lib/modules/${KVER}/build M=/src/wdt modules \
 && install -m644 stopless_wdt.ko /lib/modules/${KVER}/kernel/drivers/watchdog/stopless_wdt.ko

RUN mkdir -p /rootfs/lib/modules/${KVER} \
 && cd /lib/modules/${KVER} \
 && cp modules.builtin modules.builtin.modinfo modules.order /rootfs/lib/modules/${KVER}/ \
 && tar cf - kernel/drivers/watchdog | (cd /rootfs/lib/modules/${KVER} && tar xf -) \
 && depmod -b /rootfs ${KVER}

RUN set -e; cd /rootfs \
 && mkdir -p bin sbin usr/bin usr/sbin etc proc sys dev tmp run lib/aarch64-linux-gnu usr/lib/aarch64-linux-gnu \
 && cp -a /bin/. bin/ && cp -a /sbin/. sbin/ && cp -a /usr/bin/. usr/bin/ && cp -a /usr/sbin/. usr/sbin/ \
 && cp -a /lib/aarch64-linux-gnu/. lib/aarch64-linux-gnu/ \
 && cp -a /usr/lib/aarch64-linux-gnu/. usr/lib/aarch64-linux-gnu/ \
 && cp -a /lib/ld-linux-aarch64.so.1 lib/ \
 && cp -a /etc/passwd /etc/group /etc/nsswitch.conf /etc/hosts etc/ \
 && rm -rf usr/share/doc usr/share/man usr/share/locale \
 && printf '%s\n' '#!/bin/sh' \
      'mount -t proc none /proc' \
      'mount -t sysfs none /sys' \
      'mount -t devtmpfs none /dev 2>/dev/null' \
      'mkdir -p /dev/pts && mount -t devpts none /dev/pts 2>/dev/null' \
      'export PATH=/usr/sbin:/usr/bin:/sbin:/bin' \
      '/run.sh; echo "RUNSH_EXIT=$?"' \
      'sync; echo 1 > /proc/sys/kernel/sysrq; echo o > /proc/sysrq-trigger; sleep 5' > init \
 && chmod +x init \
 && find . | cpio -o -H newc --quiet | gzip -1 > /initrd.img \
 && cp /boot/vmlinuz-${KVER} /vmlinuz

FROM alpine:3.22
RUN apk add --no-cache qemu-system-aarch64 cpio bash
COPY --from=build /initrd.img /vm/initrd.img
COPY --from=build /vmlinuz /vm/vmlinuz
COPY runvm.sh /usr/local/bin/runvm
RUN chmod +x /usr/local/bin/runvm
ENTRYPOINT ["/usr/local/bin/runvm"]
EOF

    docker build --platform "linux/$ARCH" \
        --build-arg "KVER=$KVER" \
        -t "$WDT_IMAGE" "$context"
}

# A probe that reboots never reaches its SURVIVED line, so the verdict is read
# from the transcript rather than from an exit status.
run_watchdog() {
    local log="$BUILD/watchdog.log"
    mkdir -p "$BUILD/watchdog"
    : > "$log"
    wdt_image
    local driver probe before verdict
    for driver in softdog stopless_wdt; do
        for probe in close-no-magic close-magic freeze; do
            write_watchdog_payload "$driver" "$probe"
            before=$(wc -l < "$log")
            docker run --rm --platform "linux/$ARCH" \
                -v "$BUILD/watchdog/run.sh:/script/run.sh:ro" \
                "$WDT_IMAGE" 2>&1 | tee -a "$log"
            if tail -n +$((before + 1)) "$log" | grep -q "PROBE $driver $probe SURVIVED"; then
                verdict=survived
            else
                verdict=REBOOTED
            fi
            echo "watchdog: $driver $probe -> $verdict" >> "$BUILD/watchdog.verdict"
            echo "watchdog: $driver $probe -> $verdict"
        done
    done
}

run_tier1() {
    local log="$BUILD/tier1.log"
    : > "$log"
    write_tier1_payload
    if [ "$(uname -s)" = Linux ]; then
        KERNELINT_DIR="$BUILD/tier1" "$BUILD/tier1/run.sh" 2>&1 | tee -a "$log"
    else
        vm_run tier1
    fi
    assert_ran "$log" 5
}

run_tier2() {
    local log="$BUILD/tier2.log"
    : > "$log"
    for package in $TIER2_PACKAGES; do
        write_tier2_payload "$package"
        vm_run tier2
    done
    assert_ran "$log" 9
}

case "${1:-all}" in
tier1)
    if [ "$(uname -s)" = Linux ]; then
        compile tier1 "$(host_arch)" $TIER1_PACKAGES
    else
        compile tier1 "$ARCH" $TIER1_PACKAGES
    fi
    run_tier1
    ;;
tier2)
    compile tier2 "$ARCH" $TIER2_PACKAGES
    run_tier2
    ;;
watchdog)
    : > "$BUILD/watchdog.verdict"
    run_watchdog
    ;;
all)
    "$0" tier1
    "$0" tier2
    ;;
*)
    echo "usage: $0 [tier1|tier2|watchdog|all]" >&2
    exit 2
    ;;
esac

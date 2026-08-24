#!/bin/bash
#
# Boots the device's own rootfs userspace under riscv64 emulation and asserts
# that NanoKVM-Server binds its HTTP listener and serves the web UI. This is the
# loop that replaces a card swap: it runs the shipped binary against the shipped
# libraries, so a binary that cannot resolve libkvm.so fails here exactly as it
# fails on hardware.
#
# Usage: scripts/device-repro.sh <image.img> [NanoKVM-Server]
#
# The extracted rootfs is cached under build/repro, so once it exists the image
# argument may be "-".
#
# The second argument replaces the binary inside the extracted rootfs, which is
# how a candidate build is checked before it is flashed.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="$ROOT/build/repro"
IMAGE="${1:-}"
CANDIDATE="${2:-}"

mkdir -p "$WORK"

# The tar is only the intermediate the import consumed; once nanokvm-rvroot:repro
# exists the tar can be deleted and the harness still has its rootfs.
cached() { [ -f "$WORK/rvroot.tar.gz" ] || docker image inspect nanokvm-rvroot:repro >/dev/null 2>&1; }

if ! cached && { [ -z "$IMAGE" ] || [ ! -f "$IMAGE" ]; }; then
    echo "usage: $0 <image.img> [NanoKVM-Server]" >&2
    exit 2
fi

# Partition 2 is the ext4 rootfs, partition 1 the FAT /boot the init scripts and
# the presentation migration read their sentinels from.
if ! cached; then
    docker run --rm --privileged -v "$(cd "$(dirname "$IMAGE")" && pwd):/img:ro" -v "$WORK:/out" alpine:3 sh -euc "
        mkdir -p /mnt/r /mnt/b /rv
        start=\$(fdisk -l /img/$(basename "$IMAGE") | awk '\$1 ~ /2\$/ && \$1 ~ /img/ { print \$(NF-5) }')
        offset=\$(( start * 512 ))
        mount -o loop,offset=\$offset,ro /img/$(basename "$IMAGE") /mnt/r
        mount -o loop,offset=512,ro /img/$(basename "$IMAGE") /mnt/b
        cd /mnt/r
        tar cf - --exclude='usr/lib/python3.11' --exclude='usr/lib/tcl*' --exclude='usr/lib/firmware' \
            bin sbin lib lib64 linuxrc usr/bin usr/sbin usr/lib usr/lib64 usr/libexec kvmapp etc mnt/data mnt/cfg var root \
            | (cd /rv && tar xf -)
        mkdir -p /rv/boot && cp -a /mnt/b/. /rv/boot/
        mkdir -p /rv/proc /rv/sys /rv/dev /rv/tmp /rv/run /rv/opt /rv/media
        tar czf /out/rvroot.tar.gz -C /rv .
    "
fi

# The real libkvm.so pulls OpenCV and the MMF stack, whose static constructors
# use XTheadVector that no QEMU implements. The stub keeps the dynamic link
# realistic - it still has to be found through the binary's own runpath - while
# leaving the Go startup path untouched. It carries no DT_NEEDED so the device's
# musl loads it as-is.
if [ ! -f "$WORK/libkvm.so" ]; then
    cat > "$WORK/kvmstub.c" <<'EOF'
int kvmv_init(void){ return -1; }
int kvmv_deinit(void){ return 0; }
int kvmv_read_img(void){ return -1; }
int kvmv_hdmi_control(void){ return -1; }
int kvmv_hdmi_signal_active(void){ return 0; }
void free_kvmv_data(void){}
int set_frame_detact(void){ return 0; }
int set_h264_gop(void){ return 0; }
EOF
    docker run --rm --platform linux/riscv64 -v "$WORK:/w" -w /w riscv64/alpine:edge \
        sh -euc 'apk add --no-cache gcc musl-dev >/dev/null && gcc -shared -nostdlib -fPIC -o libkvm.so kvmstub.c'
fi

docker image inspect nanokvm-rvroot:repro >/dev/null 2>&1 || \
    docker import --platform linux/riscv64 "$WORK/rvroot.tar.gz" nanokvm-rvroot:repro >/dev/null

MOUNTS=(-v "$WORK:/repro:ro")
[ -n "$CANDIDATE" ] && MOUNTS+=(-v "$(cd "$(dirname "$CANDIDATE")" && pwd)/$(basename "$CANDIDATE"):/candidate:ro")

docker run --rm -i --platform linux/riscv64 -e QEMU_CPU=thead-c906 "${MOUNTS[@]}" nanokvm-rvroot:repro /bin/sh -s <<'EOF' 2>&1 | grep -v 'disabling zfa extension'
set -u

# What S95nanokvm does before it starts the server.
cp -r /kvmapp/server /tmp/
[ -f /candidate ] && cp /candidate /tmp/server/NanoKVM-Server
chmod +x /tmp/server/NanoKVM-Server
cp /repro/libkvm.so /tmp/server/dl_lib/libkvm.so

# libgcc_s.so.1 defines the weak __register_frame_info the binary jumps through
# before main; on the device the real libkvm.so drags it in.
LD_PRELOAD=/lib/libgcc_s.so.1 /tmp/server/NanoKVM-Server > /tmp/server.log 2>&1 &
pid=$!

waited=0
listening=0
while [ "$waited" -lt 120 ]; do
    sleep 2
    waited=$((waited + 2))
    if netstat -ltn 2>/dev/null | grep -q ':80 '; then listening=1; break; fi
    kill -0 "$pid" 2>/dev/null || break
done

echo "--- server log"
cat /tmp/server.log
echo "--- listeners"
netstat -ltn 2>/dev/null | tail -n +3

if [ "$listening" -ne 1 ]; then
    echo "FAIL: no HTTP listener after ${waited}s"
    exit 1
fi

body=$(wget -q -O - http://127.0.0.1/ 2>/dev/null | head -c 400)
case "$body" in
    *"<title>NanoKVM</title>"*) echo "OK: web UI served after ${waited}s" ;;
    *) echo "FAIL: listener bound but / did not serve the web UI"; echo "$body"; exit 1 ;;
esac

# The browser asks for the favicon while it paints the login page, so the route
# has to answer without a token. This is also the only place the real router is
# instantiated - a duplicate or conflicting route registration panics gin at
# startup, which no host-side test can reach while router imports the cgo package.
icon=$(wget -q -O - http://127.0.0.1/api/vm/favicon 2>/dev/null | wc -c)
if [ "$icon" -gt 0 ]; then
    echo "OK: favicon served unauthenticated, ${icon} bytes"
else
    echo "FAIL: /api/vm/favicon served nothing"; kill "$pid" 2>/dev/null; exit 1
fi

kill "$pid" 2>/dev/null
EOF

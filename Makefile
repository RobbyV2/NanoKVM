# Makefile for NanoKVM Project

# Configuration
UID := $(shell id -u)
GID := $(shell id -g)
# ?= so a prebuilt image can be used instead, e.g. a GHCR-cached builder in CI.
IMAGE_NAME ?= nanokvm-builder-local-$(UID)-$(GID)
PWD := $(shell pwd)
VERSION ?=
# Upstream tag the third_party/newt fork branched from; baked into the binary
# through -X main.newtVersion.
NEWT_VERSION ?= 1.16.0

# Docker run common parameters. Allocating a TTY breaks in environments without
# one, so it can be overridden with DOCKER_TTY=
DOCKER_TTY ?= -it
DOCKER_RUN_BASE := docker run -e UID=$(UID) -e GID=$(GID) -v $(PWD):/home/build/NanoKVM --rm

# Build the image with the host user's identity so entrypoint does not need
# to recursively rewrite the MaixCDK home directory at container startup.
DOCKER_BUILD_ARGS := --build-arg DOCKER_UID=$(UID) --build-arg DOCKER_GID=$(GID)

# Build commands
# C906 arch flags for the Sophgo host-tools toolchain, shared by the cgo build
# and the usb-proxy cross build.
RISCV_ARCH_FLAGS := -mcpu=c906fdv -march=rv64imafdcv0p7xthead -mcmodel=medany -mabi=lp64d
# One build path: a bare "go build" here used to diverge from build.sh and the
# release, and what it produced was packaged and flashed.
GO_BUILD_CMD := cd /home/build/NanoKVM/server && go mod tidy && ./build.sh
EDID_PROFILES_CMD := cd /home/build/NanoKVM/server && go run ../scripts/gen_edid_profiles.go
SUPPORT_BUILD_CMD := . ./home/build/MaixCDK/bin/activate && cd /home/build/NanoKVM/support/sg2002 && ./build kvm_system && ./build kvm_system add_to_kvmapp
VISION_BUILD_CMD := . ./home/build/MaixCDK/bin/activate && cd /home/build/NanoKVM/support/sg2002 && ./build kvm_vision && ./build kvm_vision add_to_kvmapp
RELEASE_BUILD_CMD := /home/build/NanoKVM/scripts/build-in-container.sh

# Tunnel binaries are staged uncompressed under build/ so package.sh can assert
# on the real ELF before the gzipped seeds ship in kvmapp/tunnels/. CARGO_HOME
# is redirected into the workspace so the registry is reachable from the host,
# which is what CI actually caches.
TUNNELS_DIR := /home/build/NanoKVM/build/tunnels
CARGO_HOME_DIR := /home/build/NanoKVM/build/cargo
WSTUNNEL_TARGET := riscv64gc-unknown-linux-musl
# opt-level "z" for size; see docs/superpowers/specs/2026-08-21-tunnels-design.md
WSTUNNEL_CARGO_CONFIG := --config profile.release.opt-level=\"z\"
# host-tools ld is binutils 2.35 and rejects the zifencei ISA attribute rustc
# emits, so link with the toolchain's own rust-lld, resolved from the sysroot.
WSTUNNEL_LLD := $$(rustc --print sysroot)/lib/rustlib/$$(rustc -vV | sed -n "s/^host: //p")/bin/rust-lld
# RUSTFLAGS overrides [build] rustflags, so uuid_unstable is carried here.
# crt-static is not default for this target; musl infers self-contained linking.
WSTUNNEL_RUSTFLAGS := --cfg uuid_unstable -C target-feature=+crt-static
WSTUNNEL_BUILD_CMD := cd /home/build/NanoKVM/third_party/wstunnel && CARGO_HOME=$(CARGO_HOME_DIR) CC_riscv64gc_unknown_linux_musl=riscv64-unknown-linux-musl-gcc CARGO_TARGET_RISCV64GC_UNKNOWN_LINUX_MUSL_LINKER="$(WSTUNNEL_LLD)" RUSTFLAGS="$(WSTUNNEL_RUSTFLAGS)" cargo $(WSTUNNEL_CARGO_CONFIG) build --release --bin wstunnel --target $(WSTUNNEL_TARGET) --no-default-features --features ring && cp -f target/$(WSTUNNEL_TARGET)/release/wstunnel $(TUNNELS_DIR)/wstunnel
NEWT_BUILD_CMD := cd /home/build/NanoKVM/third_party/newt && CGO_ENABLED=0 GOOS=linux GOARCH=riscv64 go build -trimpath -ldflags="-s -w -X main.newtVersion=$(NEWT_VERSION) -X main.newtPlatform=linux_riscv64" -o $(TUNNELS_DIR)/newt
TUNNELS_BUILD_CMD := mkdir -p $(TUNNELS_DIR) && $(WSTUNNEL_BUILD_CMD) && $(NEWT_BUILD_CMD)

# usb-proxy is C++ and links against libusb and jsoncpp, neither of which the
# builder image has a riscv64 copy of, so both are cross-built statically into
# the seed. The custom base image then needs only its musl loader.
PASSTHROUGH_DIR := /home/build/NanoKVM/build/passthrough
PASSTHROUGH_DEPS := $(PASSTHROUGH_DIR)/deps
LIBUSB_VERSION := 1.0.26
LIBUSB_SHA256 := 12ce7a61fc9854d1d2a1ffe095f7b5fac19ddba095c259e6067a46500381b5a5
LIBUSB_URL := https://github.com/libusb/libusb/releases/download/v$(LIBUSB_VERSION)/libusb-$(LIBUSB_VERSION).tar.bz2
JSONCPP_VERSION := 1.9.5
JSONCPP_SHA256 := f409856e5920c18d0c2fb85276e24ee607d2a09b5e7d5f0a371368903c275da2
JSONCPP_URL := https://github.com/open-source-parsers/jsoncpp/archive/refs/tags/$(JSONCPP_VERSION).tar.gz
LIBUSB_BUILD_CMD := cd $(PASSTHROUGH_DIR) && wget -qO libusb.tar.bz2 $(LIBUSB_URL) && echo "$(LIBUSB_SHA256)  libusb.tar.bz2" | sha256sum -c - && tar -xjf libusb.tar.bz2 && cd libusb-$(LIBUSB_VERSION) && ./configure --host=riscv64-unknown-linux-musl --prefix=$(PASSTHROUGH_DEPS) --enable-static --disable-shared --disable-udev CFLAGS="-Os $(RISCV_ARCH_FLAGS) -ffunction-sections -fdata-sections" && make -j$$(nproc) && make install
# jsoncpp ships a CMake build whose only cross input would be a toolchain file;
# three translation units archived into one library is the whole of it.
JSONCPP_BUILD_CMD := cd $(PASSTHROUGH_DIR) && wget -qO jsoncpp.tar.gz $(JSONCPP_URL) && echo "$(JSONCPP_SHA256)  jsoncpp.tar.gz" | sha256sum -c - && tar -xzf jsoncpp.tar.gz && cd jsoncpp-$(JSONCPP_VERSION) && riscv64-unknown-linux-musl-g++ -Os $(RISCV_ARCH_FLAGS) -ffunction-sections -fdata-sections -Iinclude -c src/lib_json/json_reader.cpp src/lib_json/json_value.cpp src/lib_json/json_writer.cpp && riscv64-unknown-linux-musl-ar rcs $(PASSTHROUGH_DEPS)/lib/libjsoncpp.a json_reader.o json_value.o json_writer.o && mkdir -p $(PASSTHROUGH_DEPS)/include/json && cp -a include/json/. $(PASSTHROUGH_DEPS)/include/json/
# The arch flags have to reach the link line too, which is why the fork carries
# $(CFLAGS) there: without them ld picks /lib/ld-musl-riscv64xthead.so.1 while
# the device runs /lib/ld-musl-riscv64v0p7_xthead.so.1 and nothing starts.
# PKG_CONFIG_LIBDIR confines the fork's Lua probe to the cross prefix; at its
# default it can find the host's x86 Lua and inject it into a cross link.
USB_PROXY_BUILD_CMD := cd /home/build/NanoKVM/third_party/usb-proxy && make clean && PKG_CONFIG_PATH= PKG_CONFIG_LIBDIR=$(PASSTHROUGH_DEPS)/lib/pkgconfig PKG_CONFIG_SYSROOT_DIR= CXX=riscv64-unknown-linux-musl-g++ CFLAGS="-Wall -Wextra -Os $(RISCV_ARCH_FLAGS) -ffunction-sections -fdata-sections -I$(PASSTHROUGH_DEPS)/include -I$(PASSTHROUGH_DEPS)/include/libusb-1.0" LDFLAGS="-static -Wl,--gc-sections -L$(PASSTHROUGH_DEPS)/lib" make && riscv64-unknown-linux-musl-strip usb-proxy && cp -f usb-proxy $(PASSTHROUGH_DIR)/usb-proxy
PASSTHROUGH_BUILD_CMD := rm -rf $(PASSTHROUGH_DEPS) $(PASSTHROUGH_DIR)/libusb-$(LIBUSB_VERSION) $(PASSTHROUGH_DIR)/jsoncpp-$(JSONCPP_VERSION) && mkdir -p $(PASSTHROUGH_DEPS)/include $(PASSTHROUGH_DEPS)/lib && $(LIBUSB_BUILD_CMD) && $(JSONCPP_BUILD_CMD) && $(USB_PROXY_BUILD_CMD)

.PHONY: help check-root check-version builder-image rebuild-image check-image shell app support vision \
        web tunnels passthrough edid-profiles release-build package release all clean \
        kernelint kernelint-tier1 kernelint-tier2

# Default target
all: app support

# Help target
help:
	@echo "NanoKVM Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  help          - Show this help message"
	@echo "  check-image   - Check builder Docker image and show versions"
	@echo "  builder-image - Build Docker image if not exists"
	@echo "  rebuild-image - Force rebuild Docker image"
	@echo "  shell         - Enter interactive builder environment"
	@echo "  app           - Build Go application server"
	@echo "  support       - Build kvm_system daemon"
	@echo "  vision        - Build video libraries (libkvm.so)"
	@echo "  web           - Build the frontend into web/dist"
	@echo "  tunnels       - Build wstunnel + newt seeds into kvmapp/tunnels"
	@echo "  passthrough   - Build the usb-proxy seed into kvmapp/passthrough"
	@echo "  edid-profiles - Regenerate the shipped EDID profile table"
	@echo "  kernelint-tier1 - Kernel tests needing netns and vhci_hcd"
	@echo "  kernelint-tier2 - Kernel tests needing a UDC, in a QEMU VM"
	@echo "  kernelint     - Both kernel test tiers"
	@echo "  all           - Build both app and support (default)"
	@echo "  release-build - Build every riscv64 release artifact in one pass"
	@echo "  package       - Assemble nanokvm_<VERSION>.tar.gz + latest.json"
	@echo "  release       - release-build + web + package (needs VERSION=x.y.z)"
	@echo "  clean         - Clean build artifacts"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - Docker must be installed and running"
	@echo "  - Must not run as root user"

# Packaging takes tens of minutes of cross-compilation before it reaches
# package.sh, so the targets that need VERSION assert it before any of that.
check-version:
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION is required, e.g. make package VERSION=2.4.4"; \
		exit 1; \
	fi

# Security check - prevent running as root
check-root:
	@if [ "$$(id -u)" -eq 0 ]; then \
		echo "Can't run as root"; \
		exit 1; \
	fi

# Check if builder image exists and show versions
check-image: check-root
	@echo "Checking builder image..."
	@echo "Golang version: " && \
		docker run --rm -i $(IMAGE_NAME) go version && \
		echo "" && \
		echo "Host-tools version:" && \
		docker run --rm -i $(IMAGE_NAME) riscv64-unknown-linux-musl-gcc -v && \
		echo ""

# Build Docker image if it doesn't exist
builder-image: check-root
	@if ! docker image inspect $(IMAGE_NAME) >/dev/null 2>&1; then \
		echo "Building Docker image..."; \
		docker build $(DOCKER_BUILD_ARGS) -t $(IMAGE_NAME) -f docker/Dockerfile ./; \
	else \
		echo "Docker image $(IMAGE_NAME) already exists."; \
	fi

# Force rebuild Docker image
rebuild-image: check-root
	@echo "Force rebuilding Docker image..."
	@docker build --no-cache $(DOCKER_BUILD_ARGS) -t $(IMAGE_NAME) -f docker/Dockerfile ./

# Enter interactive shell (equivalent to build.sh with no arguments)
shell: check-root builder-image
	@echo "Switching into builder..."
	@$(DOCKER_RUN_BASE) -it $(IMAGE_NAME) /bin/bash -c ". ./home/build/MaixCDK/bin/activate && cd /home/build/NanoKVM ; exec bash"

# Build Go application
app: check-root builder-image
	@echo "Building app..."
	@$(DOCKER_RUN_BASE) $(DOCKER_TTY) $(IMAGE_NAME) /bin/bash -c '$(GO_BUILD_CMD)'

# Build hardware support libraries
support: check-root builder-image
	@echo "Building support..."
	@$(DOCKER_RUN_BASE) $(DOCKER_TTY) $(IMAGE_NAME) /bin/bash -c '$(SUPPORT_BUILD_CMD)'

# Build video libraries (libkvm.so) into kvmapp/server/dl_lib
vision: check-root builder-image
	@echo "Building vision..."
	@$(DOCKER_RUN_BASE) $(DOCKER_TTY) $(IMAGE_NAME) /bin/bash -c '$(VISION_BUILD_CMD)'

# Build every riscv64 release artifact in one container pass
release-build: check-root builder-image
	@echo "Building release artifacts..."
	@$(DOCKER_RUN_BASE) $(DOCKER_TTY) $(IMAGE_NAME) /bin/bash -c '$(RELEASE_BUILD_CMD)'

# Regenerate server/service/edid/profiles_gen.go from the pinned linuxhw/EDID
# commit named in scripts/gen_edid_profiles.go. Needs network.
edid-profiles: check-root builder-image
	@echo "Generating EDID profiles..."
	@$(DOCKER_RUN_BASE) $(DOCKER_TTY) $(IMAGE_NAME) /bin/bash -c '$(EDID_PROFILES_CMD)'

# The //go:build kernelint tests. Tier 1 needs a network namespace and
# vhci_hcd; tier 2 needs dummy_hcd and boots a VM for it. See scripts/kernelint.sh.
kernelint-tier1:
	@scripts/kernelint.sh tier1

kernelint-tier2:
	@scripts/kernelint.sh tier2

kernelint: kernelint-tier1 kernelint-tier2

# Build the frontend (runs on the host; the builder image has no Node)
web:
	@echo "Building web..."
	@cd web && pnpm install --frozen-lockfile && pnpm build

# Cross-build the tunnel binaries and seed them into kvmapp/tunnels.
# gzip -n is required: package.sh derives SOURCE_DATE_EPOCH from the commit date
# and an embedded gzip timestamp would break reproducibility.
tunnels: check-root builder-image
	@echo "Building tunnels..."
	@$(DOCKER_RUN_BASE) $(DOCKER_TTY) $(IMAGE_NAME) /bin/bash -c '$(TUNNELS_BUILD_CMD)'
	@mkdir -p kvmapp/tunnels
	@gzip -9 -n -c build/tunnels/wstunnel > kvmapp/tunnels/wstunnel.gz
	@gzip -9 -n -c build/tunnels/newt > kvmapp/tunnels/newt.gz

# Cross-build usb-proxy and seed it into kvmapp/passthrough, on the same terms
# as the tunnels: staged uncompressed under build/ for package.sh's arch check,
# shipped gzipped, and gzip -n for the same reproducibility reason.
passthrough: check-root builder-image
	@echo "Building usb-proxy..."
	@$(DOCKER_RUN_BASE) $(DOCKER_TTY) $(IMAGE_NAME) /bin/bash -c '$(PASSTHROUGH_BUILD_CMD)'
	@mkdir -p kvmapp/passthrough
	@gzip -9 -n -c build/passthrough/usb-proxy > kvmapp/passthrough/usb-proxy.gz

# Assemble the release package and its manifest
package: check-version tunnels passthrough
	@scripts/package.sh "$(VERSION)"

# Full local release: riscv64 artifacts + frontend + package.
# Invoked as sub-makes rather than prerequisites: packaging asserts on what the
# earlier steps produce, so the order matters even under make -j.
release:
	@$(MAKE) check-version
	@$(MAKE) release-build
	@$(MAKE) web
	@$(MAKE) package VERSION="$(VERSION)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@if [ -f server/NanoKVM-Server ]; then \
		rm -f server/NanoKVM-Server; \
		echo "Removed server/NanoKVM-Server"; \
	fi
	@if [ -d support/sg2002/build ]; then \
		rm -rf support/sg2002/build; \
		echo "Removed support/sg2002/build"; \
	fi
	@if [ -d build/release ]; then \
		rm -rf build/release; \
		echo "Removed build/release"; \
	fi
	@if [ -d web/dist ]; then \
		rm -rf web/dist; \
		echo "Removed web/dist"; \
	fi
	@echo "Clean completed."

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
GO_BUILD_CMD := cd /home/build/NanoKVM/server && go mod tidy && CGO_ENABLED=1 GOOS=linux GOARCH=riscv64 CC=riscv64-unknown-linux-musl-gcc CGO_CFLAGS="-mcpu=c906fdv -march=rv64imafdcv0p7xthead -mcmodel=medany -mabi=lp64d" go build
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

.PHONY: help check-root builder-image rebuild-image check-image shell app support vision \
        web tunnels edid-profiles release-build package release all clean

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
	@echo "  edid-profiles - Regenerate the shipped EDID profile table"
	@echo "  all           - Build both app and support (default)"
	@echo "  release-build - Build every riscv64 release artifact in one pass"
	@echo "  package       - Assemble nanokvm_<VERSION>.tar.gz + latest.json"
	@echo "  release       - release-build + web + package (needs VERSION=x.y.z)"
	@echo "  clean         - Clean build artifacts"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - Docker must be installed and running"
	@echo "  - Must not run as root user"

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

# Assemble the release package and its manifest
package: tunnels
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION is required, e.g. make package VERSION=2.4.4"; \
		exit 1; \
	fi
	@scripts/package.sh "$(VERSION)"

# Full local release: riscv64 artifacts + frontend + package.
# Invoked as sub-makes rather than prerequisites: packaging asserts on what the
# earlier steps produce, so the order matters even under make -j.
release:
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

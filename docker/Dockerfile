FROM ubuntu:24.04 AS base_apt

# RUN echo 'Acquire::http::Proxy "http://cache.lan:3142";' > /etc/apt/apt.conf.d/00cacher
RUN userdel -r ubuntu
RUN apt update \
    && apt install wget git cmake build-essential python3 python3-venv python3-pip autoconf automake libtool gosu patchelf -y

ARG DOCKER_USER=build
ARG DOCKER_UID=1000
ARG DOCKER_GID=1000
# Reuse existing accounts for numeric IDs that are already present in the
# base image. All later ownership and user switches use numeric IDs, so a
# dedicated build user/group is only needed when the ID does not exist.
RUN set -eux; \
    if ! getent group "$DOCKER_GID" >/dev/null; then \
        groupadd --gid "$DOCKER_GID" "$DOCKER_USER"; \
    fi; \
    if ! getent passwd "$DOCKER_UID" >/dev/null; then \
        useradd --uid "$DOCKER_UID" --gid "$DOCKER_GID" --no-create-home \
            --home-dir "/home/$DOCKER_USER" --shell /bin/bash "$DOCKER_USER"; \
    fi; \
    install -d -o "$DOCKER_UID" -g "$DOCKER_GID" "/home/$DOCKER_USER"
USER $DOCKER_UID:$DOCKER_GID

FROM base_apt AS host_tools

USER root
RUN wget -O /tmp/host-tools.tar.gz https://sophon-file.sophon.cn/sophon-prod-s3/drive/23/03/07/16/host-tools.tar.gz \
    && tar -C /usr/local -zxvf /tmp/host-tools.tar.gz  \
    && rm -f /tmp/host-tools.tar.gz \
    && export PATH=$PATH:/usr/local/host-tools/gcc/riscv64-linux-musl-x86_64/bin \
    && echo "Installed host-tools gcc" && riscv64-unknown-linux-musl-gcc -v

FROM base_apt AS golang

USER root
RUN bash -c 'wget -O /tmp/go.tar.gz https://go.dev/dl/go1.25.0.linux-$([ `uname -m` == "x86_64" ] && echo "amd64" || uname -m).tar.gz' \
    && tar -C /usr/local -zxvf /tmp/go.tar.gz \
    && export PATH=$PATH:/usr/local/go/bin \
    && echo "Installed golang" &&  go version

COPY server /tmp/go_cache

ENV PATH="$PATH:/usr/local/go/bin"

RUN cd /tmp/go_cache \
    && echo "Caching go dep" \
    && go mod download \
    && go mod tidy \
    && cd / \
    && rm -rf /tmp/go_cache

FROM base_apt AS rust

USER root
ARG RUST_VERSION=1.97.0
ENV RUSTUP_HOME="/usr/local/rustup"
ENV CARGO_HOME="/usr/local/cargo"
RUN wget -O /tmp/rustup-init.sh https://sh.rustup.rs \
    && sh /tmp/rustup-init.sh -y --no-modify-path --profile minimal --default-toolchain "$RUST_VERSION" \
    && rm -f /tmp/rustup-init.sh \
    && export PATH=$PATH:$CARGO_HOME/bin \
    && rustup target add riscv64gc-unknown-linux-musl \
    && echo "Installed rust" && cargo --version && rustc --version

FROM base_apt AS sdk

ENV HOME="/home/$DOCKER_USER"

RUN cd ~ \
    && git clone https://github.com/Sipeed/MaixCDK \
    && cd MaixCDK \
    && python3 -m venv . \
    && . ./bin/activate \
    && pip install -U -r requirements.txt 

COPY --chown=$DOCKER_UID:$DOCKER_GID . /home/$DOCKER_USER/NanoKVM

RUN cd ~/MaixCDK \
    && . ./bin/activate \
    && cd ~/NanoKVM/support/sg2002 \
    && ./build kvm_system

FROM base_apt

# Install golang
COPY --from=golang /usr/local/go /usr/local/go
COPY --chown=$DOCKER_UID:$DOCKER_GID --from=golang /root/go /home/$DOCKER_USER/go
ENV PATH="$PATH:/usr/local/go/bin"

# Install host tools
COPY --from=host_tools /usr/local/host-tools /usr/local/host-tools
ENV PATH="$PATH:/usr/local/host-tools/gcc/riscv64-linux-musl-x86_64/bin"

# Install rust. The toolchain itself stays read-only under /usr/local, while
# CARGO_HOME lands in the build user's home so cargo can write its registry.
COPY --from=rust /usr/local/rustup /usr/local/rustup
COPY --chown=$DOCKER_UID:$DOCKER_GID --from=rust /usr/local/cargo /home/$DOCKER_USER/.cargo
ENV RUSTUP_HOME="/usr/local/rustup"
ENV CARGO_HOME="/home/$DOCKER_USER/.cargo"
ENV PATH="$PATH:/home/$DOCKER_USER/.cargo/bin"

# Install SDK
COPY --chown=$DOCKER_UID:$DOCKER_GID --from=sdk /home/$DOCKER_USER/MaixCDK /home/$DOCKER_USER/MaixCDK
COPY --chmod=+x docker/entrypoint /entrypoint

RUN echo "Verify build tools" \
    && echo "Installed host-tools gcc" && riscv64-unknown-linux-musl-gcc -v \
    && echo "Installed golang" && go version \
    && echo "Installed rust" && cargo --version \
    && rustup target list --installed | grep -qx riscv64gc-unknown-linux-musl

USER root
RUN chmod 0755 /entrypoint \
 && sed "s/\$DOCKER_USER/$DOCKER_USER/g" -i /entrypoint

ENTRYPOINT ["/entrypoint"]

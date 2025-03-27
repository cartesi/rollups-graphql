# (c) Cartesi and individual authors (see AUTHORS)
# SPDX-License-Identifier: Apache-2.0 (see LICENSE)

# syntax=docker.io/docker/dockerfile:1

FROM debian:bookworm-20250113-slim AS rollups-node

ARG NODE_RUNTIME_DIR=/var/lib/cartesi-rollups-node
ARG TARGETARCH=arm64

USER root

# Download system dependencies required at runtime.
ARG DEBIAN_FRONTEND=noninteractive
RUN <<EOF
    set -e
    apt-get update
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        libboost-all-dev \
        libslirp0 \
        lua5.4 \
        procps \
        tini
    rm -rf /var/lib/apt/lists/*
    mkdir -p ${NODE_RUNTIME_DIR}/snapshots ${NODE_RUNTIME_DIR}/data
EOF

# Copy Go binary.
ARG GO_BUILD_PATH

# Install cartesi machine
ENV CARTESI_MACHINE_VERSION=0.19.0
RUN <<EOF
    set -e
    curl -LO https://github.com/cartesi/machine-emulator/releases/download/v0.19.0-alpha3/cartesi-machine-v${CARTESI_MACHINE_VERSION}_${TARGETARCH}.deb
    dpkg -i ./cartesi-machine-v${CARTESI_MACHINE_VERSION}_${TARGETARCH}.deb
    rm ./cartesi-machine-v${CARTESI_MACHINE_VERSION}_${TARGETARCH}.deb
EOF

RUN curl -L https://github.com/cartesi/image-kernel/releases/download/v0.20.0/linux-6.5.13-ctsi-1-v0.20.0.bin -o /usr/share/cartesi-machine/images/linux.bin

RUN curl -L https://github.com/cartesi/machine-emulator-tools/releases/download/v0.16.1/rootfs-tools-v0.16.1.ext2 -o /usr/share/cartesi-machine/images/rootfs.ext2

# Install rollups node
RUN <<EOF
    set -e
    curl -LO https://github.com/cartesi/rollups-node/releases/download/v2.0.0-alpha.2/cartesi-rollups-node-v2.0.0-alpha.2_${TARGETARCH}.deb
    dpkg -i ./cartesi-rollups-node-v2.0.0-alpha.2_${TARGETARCH}.deb
    rm ./cartesi-rollups-node-v2.0.0-alpha.2_${TARGETARCH}.deb
EOF

WORKDIR ${NODE_RUNTIME_DIR}

HEALTHCHECK --interval=1s --timeout=1s --retries=5 \
    CMD curl -G -f -H 'Content-Type: application/json' http://127.0.0.1:10000/healthz

RUN mkdir applications
RUN cartesi-machine --ram-length=128Mi --store=applications/echo-dapp --final-hash -- ioctl-echo-loop --vouchers=1 --notices=1 --reports=1 --verbose=1

COPY --chown=cartesi:cartesi ./ci/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
CMD ["tini", "--", "/entrypoint.sh"]

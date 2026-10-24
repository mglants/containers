# syntax=docker/dockerfile:1.7
# Source-built BigBlueButton signaling; version and target are selected by Bake.
ARG VERSION
# renovate: datasource=github-tags depName=bigbluebutton/bbb-webrtc-sfu
ARG SIGNALING_VERSION=v2.25.0-beta.3

FROM ubuntu:24.04@sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac AS sources
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*
ARG SIGNALING_VERSION
ADD https://codeload.github.com/bigbluebutton/bbb-webrtc-sfu/tar.gz/refs/tags/${SIGNALING_VERSION} /tmp/signaling.tar
RUN mkdir -p /sources/signaling && tar --no-same-owner -xf /tmp/signaling.tar --strip-components=1 -C /sources/signaling && rm /tmp/signaling.tar

FROM golang:1.27-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -ldflags="-s -w" -o /out/bbb-runtime .

FROM node:24-bookworm-slim@sha256:55fb5f5a527ea2c5c4a95f963e48ef90aa38c7b346054fdc8ee9957a2b55d205 AS node-builder
ARG VERSION
ENV NODE_OPTIONS=--max-old-space-size=6144
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git python3 make g++ pkg-config && rm -rf /var/lib/apt/lists/*
COPY --from=sources /sources/signaling/ /sources/signaling/
COPY build/build-node.sh /build/build-node.sh

FROM node-builder AS build-signaling
RUN apt-get update && apt-get install -y --no-install-recommends python3-pip python3-venv \
    && rm -rf /var/lib/apt/lists/*
RUN --mount=type=cache,id=bbb-npm,target=/root/.npm,sharing=locked bash /build/build-node.sh signaling

FROM build-signaling AS staged-signaling
RUN rm -rf /out/.git /out/node_modules/mediasoup/worker/out/Release/subprojects \
    /out/node_modules/mediasoup/worker/out/Release/mediasoup-worker.p \
    /out/node_modules/mediasoup/worker/out/Release/deps

# Runtime needs Node itself; npm, Yarn and development headers stay in builders.
FROM node:24-bookworm-slim@sha256:55fb5f5a527ea2c5c4a95f963e48ef90aa38c7b346054fdc8ee9957a2b55d205 AS node-binary

FROM debian:bookworm-slim@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587 AS node-runtime
COPY --from=node-binary /usr/local/bin/node /usr/local/bin/node
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tini libstdc++6 libatomic1 \
    && rm -rf /var/lib/apt/lists/* && groupadd -g 10001 bbb && useradd -u 10001 -g bbb -M -d /tmp bbb
ENV NODE_ENV=production HOME=/tmp LANG=C.UTF-8
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--"]

FROM node-runtime AS signaling
WORKDIR /usr/local/bigbluebutton/bbb-webrtc-sfu
COPY --from=staged-signaling /out/ ./
ENV NODE_CONFIG_DIR=/usr/local/bigbluebutton/bbb-webrtc-sfu/config:/etc/bigbluebutton/bbb-webrtc-sfu
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --from=node-binary /usr/local/LICENSE /usr/local/share/licenses/node/LICENSE
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh signaling
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "signaling"]
CMD ["node", "server.js"]

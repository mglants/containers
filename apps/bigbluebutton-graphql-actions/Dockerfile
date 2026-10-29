# syntax=docker/dockerfile:1.7
# Source-built BigBlueButton graphql-actions; version and target are selected by Bake.
ARG VERSION

FROM ubuntu:26.04@sha256:6a06111c3b0a8600feadb270aec771dc239bf407b25625f93f44ab00e11f4d3c AS sources
ARG VERSION
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*
ADD https://codeload.github.com/bigbluebutton/bigbluebutton/tar.gz/refs/tags/v${VERSION} /tmp/bbb.tar
RUN mkdir -p /sources/bbb && tar --no-same-owner -xf /tmp/bbb.tar --strip-components=1 -C /sources/bbb && rm /tmp/bbb.tar

FROM golang:1.27-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -ldflags="-s -w" -o /out/bbb-runtime .

FROM node:24-bookworm-slim@sha256:55fb5f5a527ea2c5c4a95f963e48ef90aa38c7b346054fdc8ee9957a2b55d205 AS node-builder
ARG VERSION
ENV NODE_OPTIONS=--max-old-space-size=6144
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git python3 make g++ pkg-config && rm -rf /var/lib/apt/lists/*
COPY --from=sources /sources/bbb/ /sources/bbb/
COPY build/build-node.sh /build/build-node.sh

FROM node-builder AS build-actions
COPY build/patch-actions.py /build/patch-actions.py
RUN cd /sources/bbb/bbb-graphql-actions && python3 /build/patch-actions.py
RUN --mount=type=cache,id=bbb-npm,target=/root/.npm,sharing=locked bash /build/build-node.sh graphql-actions

# Runtime needs Node itself; npm, Yarn and development headers stay in builders.
FROM node:24-bookworm-slim@sha256:55fb5f5a527ea2c5c4a95f963e48ef90aa38c7b346054fdc8ee9957a2b55d205 AS node-binary

FROM debian:bookworm-slim@sha256:7ff6ba31527b9e336162fa061746c366c4b2032f4d2cff2078bea43f78efe3f8 AS node-runtime
COPY --from=node-binary /usr/local/bin/node /usr/local/bin/node
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tini libstdc++6 libatomic1 \
    && rm -rf /var/lib/apt/lists/* && groupadd -g 10001 bbb && useradd -u 10001 -g bbb -M -d /tmp bbb
ENV NODE_ENV=production HOME=/tmp LANG=C.UTF-8
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--"]

FROM node-runtime AS graphql-actions
WORKDIR /usr/local/bigbluebutton/bbb-graphql-actions
COPY --from=build-actions /out/ ./
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --from=node-binary /usr/local/LICENSE /usr/local/share/licenses/node/LICENSE
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "graphql-actions"]
CMD ["node", "bbb-graphql-actions.js"]

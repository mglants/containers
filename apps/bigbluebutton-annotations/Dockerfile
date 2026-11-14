# syntax=docker/dockerfile:1.28
# Source-built BigBlueButton annotations; version and target are selected by Bake.
ARG VERSION

FROM ubuntu:24.04@sha256:24737cef17f4e22b1977df7d9a05e77c4e28b7da5b8911f8d8ccda136bf3573c AS sources
ARG VERSION
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*
ADD https://codeload.github.com/bigbluebutton/bigbluebutton/tar.gz/refs/tags/v${VERSION} /tmp/bbb.tar
RUN mkdir -p /sources/bbb && tar --no-same-owner -xf /tmp/bbb.tar --strip-components=1 -C /sources/bbb && rm /tmp/bbb.tar

FROM golang:1.27-bookworm@sha256:5c3e9b43cfd3bb95c7937462e29d87ada57b481a96fa23ad0eb61778d7a00110 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -ldflags="-s -w" -o /out/bbb-runtime .

FROM sources AS runtime-defaults
# Inline Python content is supplied by the Dockerfile, not the build context.
# droast ignore=DF077
COPY <<'PYCODE' /tmp/bbb-build-2.py
from pathlib import Path
out=Path('/runtime-defaults');out.mkdir()
(out/'annotations.json').write_text(Path('/sources/bbb/bbb-export-annotations/config/settings.json').read_text())
PYCODE
RUN python3 /tmp/bbb-build-2.py && rm /tmp/bbb-build-2.py

FROM node:24-bookworm-slim@sha256:c86ecead9e55a5210ac1727de38a4ca840bf03dcaf34dc9749addbf383b1920c AS node-builder
ARG VERSION
ENV NODE_OPTIONS=--max-old-space-size=6144
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git python3 make g++ pkg-config && rm -rf /var/lib/apt/lists/*
COPY --from=sources /sources/bbb/ /sources/bbb/
COPY build/build-node.sh /build/build-node.sh

FROM node-builder AS build-annotations
RUN --mount=type=cache,id=bbb-npm,target=/root/.npm,sharing=locked bash /build/build-node.sh annotations

# Runtime needs Node itself; npm, Yarn and development headers stay in builders.
FROM node:24-bookworm-slim@sha256:c86ecead9e55a5210ac1727de38a4ca840bf03dcaf34dc9749addbf383b1920c AS node-binary

FROM debian:bookworm-slim@sha256:7ff6ba31527b9e336162fa061746c366c4b2032f4d2cff2078bea43f78efe3f8 AS node-runtime
COPY --from=node-binary /usr/local/bin/node /usr/local/bin/node
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tini libstdc++6 libatomic1 \
    && rm -rf /var/lib/apt/lists/* && groupadd -g 10001 bbb && useradd -u 10001 -g bbb -M -d /tmp bbb
ENV NODE_ENV=production HOME=/tmp LANG=C.UTF-8
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--"]

FROM node-runtime AS annotations
USER root
RUN apt-get update && apt-get install -y --no-install-recommends cairosvg ghostscript imagemagick poppler-utils \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /usr/local/bigbluebutton/bbb-export-annotations
COPY --from=build-annotations /out/ ./
COPY --from=sources /sources/bbb/build/packages-template/bbb-export-annotations/fonts/ /usr/local/share/fonts/
COPY --from=sources /sources/bbb/bigbluebutton-config/slides/ /usr/share/bigbluebutton/blank/
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --from=node-binary /usr/local/LICENSE /usr/local/share/licenses/node/LICENSE
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh annotations
COPY --from=runtime-defaults /runtime-defaults/ /usr/share/bbb-container/defaults/
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "annotations"]
CMD ["node", "master.js"]

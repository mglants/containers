# syntax=docker/dockerfile:1.7
# Source-built BigBlueButton annotations; version and target are selected by Bake.
ARG VERSION

FROM ubuntu:24.04@sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac AS sources
ARG VERSION
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*
ADD https://codeload.github.com/bigbluebutton/bigbluebutton/tar.gz/refs/tags/v${VERSION} /tmp/bbb.tar
RUN mkdir -p /sources/bbb && tar --no-same-owner -xf /tmp/bbb.tar --strip-components=1 -C /sources/bbb && rm /tmp/bbb.tar

FROM golang:1.27-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -o /out/bbb-runtime .

FROM sources AS runtime-defaults
# Inline Python content is supplied by the Dockerfile, not the build context.
# droast ignore=DF077
COPY <<'PYCODE' /tmp/bbb-build-2.py
from pathlib import Path
out=Path('/runtime-defaults');out.mkdir()
(out/'annotations.json').write_text(Path('/sources/bbb/bbb-export-annotations/config/settings.json').read_text())
PYCODE
RUN python3 /tmp/bbb-build-2.py && rm /tmp/bbb-build-2.py

FROM node:24-bookworm-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5 AS node-builder
ARG VERSION
ENV NODE_OPTIONS=--max-old-space-size=6144
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git python3 make g++ pkg-config && rm -rf /var/lib/apt/lists/*
COPY --from=sources /sources/bbb/ /sources/bbb/
COPY build/build-node.sh /build/build-node.sh

FROM node-builder AS build-annotations
RUN --mount=type=cache,id=bbb-npm,target=/root/.npm,sharing=locked bash /build/build-node.sh annotations

FROM node:24-bookworm-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5 AS node-runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tini \
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
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh annotations
COPY --from=runtime-defaults /runtime-defaults/ /usr/share/bbb-container/defaults/
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "annotations"]
CMD ["node", "master.js"]

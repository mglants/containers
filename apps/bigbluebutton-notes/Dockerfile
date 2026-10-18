# syntax=docker/dockerfile:1.7
# Source-built BigBlueButton notes; version and target are selected by Bake.
ARG VERSION

FROM ubuntu:26.04@sha256:c098726dd9b9733f3e885f8de0ef375206a311b0a317eadc40d347b1115fb944 AS sources
ARG VERSION
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*
ADD https://codeload.github.com/bigbluebutton/bigbluebutton/tar.gz/refs/tags/v${VERSION} /tmp/bbb.tar
RUN mkdir -p /sources/bbb && tar --no-same-owner -xf /tmp/bbb.tar --strip-components=1 -C /sources/bbb && rm /tmp/bbb.tar

FROM golang:1.27-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -ldflags="-s -w" -o /out/bbb-runtime .

FROM node:24-bookworm-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5 AS node-builder
ARG VERSION
ENV NODE_OPTIONS=--max-old-space-size=6144
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git python3 make g++ pkg-config && rm -rf /var/lib/apt/lists/*
COPY --from=sources /sources/bbb/ /sources/bbb/
COPY build/build-node.sh /build/build-node.sh

FROM node-builder AS build-notes
RUN --mount=type=cache,id=bbb-npm,target=/root/.npm,sharing=locked bash /build/build-node.sh notes

# Runtime needs Node itself; npm, Yarn and development headers stay in builders.
FROM node:24-bookworm-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5 AS node-binary

FROM debian:bookworm-slim@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587 AS node-runtime
COPY --from=node-binary /usr/local/bin/node /usr/local/bin/node
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tini libstdc++6 libatomic1 \
    && rm -rf /var/lib/apt/lists/* && groupadd -g 10001 bbb && useradd -u 10001 -g bbb -M -d /tmp bbb
ENV NODE_ENV=production HOME=/tmp LANG=C.UTF-8
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--"]

FROM node-runtime AS notes
USER root
RUN apt-get update && apt-get install -y --no-install-recommends wkhtmltopdf xvfb xauth fonts-dejavu \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /usr/share/bbb-shared-notes-server
COPY --from=build-notes /out/ ./
COPY --chmod=755 runtime/run-command.sh ./run-in-systemd.sh
COPY --chmod=755 runtime/wkhtmltopdf.sh /usr/local/bin/wkhtmltopdf
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --from=node-binary /usr/local/LICENSE /usr/local/share/licenses/node/LICENSE
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh notes
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "notes"]
CMD ["node", "bbb-shared-notes-server.js"]

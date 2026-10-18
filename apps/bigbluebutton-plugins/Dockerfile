# syntax=docker/dockerfile:1.7
# Source-built BigBlueButton plugins; version and target are selected by Bake.
ARG VERSION
# renovate: datasource=github-tags depName=bigbluebutton/bbb-plugin-live-transcription
ARG TRANSCRIPTION_PLUGIN_VERSION=v0.1.2

FROM golang:1.27-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -ldflags="-s -w" -o /out/bbb-runtime .

FROM ubuntu:24.04@sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac AS addon-sources
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*
ARG TRANSCRIPTION_PLUGIN_VERSION
ADD https://codeload.github.com/bigbluebutton/bbb-plugin-live-transcription/tar.gz/refs/tags/${TRANSCRIPTION_PLUGIN_VERSION} /tmp/transcription-plugin.tar
RUN mkdir -p /sources/transcription-plugin && tar --no-same-owner -xf /tmp/transcription-plugin.tar --strip-components=1 -C /sources/transcription-plugin && rm /tmp/transcription-plugin.tar

FROM node:24-bookworm-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5 AS build-plugins
WORKDIR /app
COPY --from=addon-sources /sources/transcription-plugin/ ./
RUN --mount=type=cache,id=bbb-npm-addons,target=/root/.npm npm ci --no-audit --no-fund && npm run build-bundle

FROM nginx:1.31-alpine@sha256:892bb2152464f0c5d8972593a5f5afaf8054552580b699d95f8ff4af08c2d33d AS plugins
COPY --from=build-plugins /app/dist/ /usr/share/nginx/html/live-transcription/
COPY build/plugins.conf /etc/nginx/nginx.conf
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh plugins
USER nginx
ENTRYPOINT ["/usr/local/bin/bbb-runtime", "plugins"]
CMD ["nginx", "-g", "daemon off;"]

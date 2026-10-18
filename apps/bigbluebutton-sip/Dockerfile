# syntax=docker/dockerfile:1.7
# Source-built BigBlueButton sip; version and target are selected by Bake.
ARG VERSION
# renovate: datasource=github-tags depName=livekit/sip
ARG SIP_VERSION=v1.20.0

FROM ubuntu:24.04@sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac AS source-tools
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*

FROM golang:1.27-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -ldflags="-s -w" -o /out/bbb-runtime .

FROM source-tools AS addon-sources
ARG SIP_VERSION
ADD https://codeload.github.com/livekit/sip/tar.gz/refs/tags/${SIP_VERSION} /tmp/sip.tar
RUN mkdir -p /sources/sip && tar --no-same-owner -xf /tmp/sip.tar --strip-components=1 -C /sources/sip && rm /tmp/sip.tar

FROM golang:1.27-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS build-sip
ARG SIP_VERSION
ENV GOTOOLCHAIN=local
RUN apt-get update && apt-get install -y --no-install-recommends pkg-config libopus-dev libopusfile-dev libsoxr-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=addon-sources /sources/sip/ ./
COPY build/patches/no-pin-sub_sip.patch /build/no-pin-sub_sip.patch
RUN git apply /build/no-pin-sub_sip.patch
RUN --mount=type=cache,id=bbb-go-mod,target=/go/pkg/mod --mount=type=cache,id=bbb-go-build,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -mod=readonly -trimpath -ldflags="-s -w -X github.com/livekit/sip/version.Version=${SIP_VERSION}" -o /out/livekit-sip ./cmd/livekit-sip

FROM debian:bookworm-slim@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587 AS sip
RUN apt-get update && apt-get install -y --no-install-recommends tini ca-certificates libopus0 libopusfile0 libsoxr0 && rm -rf /var/lib/apt/lists/*
COPY --from=build-sip /out/livekit-sip /usr/local/bin/livekit-sip
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh sip
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "sip"]
CMD ["livekit-sip", "--config", "/etc/bigbluebutton/livekit-sip.yaml"]

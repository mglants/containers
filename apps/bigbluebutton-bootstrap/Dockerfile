# syntax=docker/dockerfile:1.7
# Source-built BigBlueButton bootstrap; version and target are selected by Bake.
ARG VERSION
# renovate: datasource=github-releases depName=hasura/graphql-engine
ARG HASURA_VERSION=v2.50.3

FROM ubuntu:24.04@sha256:24737cef17f4e22b1977df7d9a05e77c4e28b7da5b8911f8d8ccda136bf3573c AS bbb-source
ARG VERSION
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*
ADD https://codeload.github.com/bigbluebutton/bigbluebutton/tar.gz/refs/tags/v${VERSION} /tmp/bbb.tar
RUN mkdir -p /sources/bbb && tar --no-same-owner -xf /tmp/bbb.tar --strip-components=1 -C /sources/bbb && rm /tmp/bbb.tar

FROM bbb-source AS sources
ARG HASURA_VERSION
ADD https://github.com/hasura/graphql-engine/releases/download/${HASURA_VERSION}/cli-hasura-linux-amd64 /tools/hasura
RUN chmod 755 /tools/hasura

FROM golang:1.27-bookworm@sha256:d9b6a1beb9474adc2a1ce842573639b44b2a0dd49b6669f51d3821baf9ec37c9 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -ldflags="-s -w" -o /out/bbb-runtime .

FROM ubuntu:24.04@sha256:24737cef17f4e22b1977df7d9a05e77c4e28b7da5b8911f8d8ccda136bf3573c AS minimal-runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tini \
    && rm -rf /var/lib/apt/lists/* && groupadd -g 10001 bbb && useradd -u 10001 -g bbb -M -d /tmp bbb
ENV HOME=/tmp LANG=C.UTF-8
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--"]

FROM minimal-runtime AS bootstrap
USER root
RUN apt-get update && apt-get install -y --no-install-recommends postgresql-client-16 python3 python3-yaml && rm -rf /var/lib/apt/lists/*
COPY --from=sources /tools/hasura /usr/local/bin/hasura
COPY --from=sources /sources/bbb/bbb-graphql-server/ /usr/share/bbb-graphql-server/
COPY --from=sources /sources/bbb/bbb-shared-notes-server/blocknote_schema.sql /usr/share/bbb-container/blocknote_schema.sql
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --chmod=755 runtime/bootstrap.py /usr/local/bin/bbb-bootstrap
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "bootstrap"]
CMD ["hasura", "version", "--skip-update-check"]

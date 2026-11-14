# syntax=docker/dockerfile:1.7
# Source-built BigBlueButton graphql-middleware; version and target are selected by Bake.
ARG VERSION

FROM ubuntu:24.04@sha256:fe81560990fae86dc4743202b2cb4d3df98d0bcf949173cecef15ce1127ef852 AS sources
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

FROM golang:1.27-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5 AS build-middleware
ENV GOTOOLCHAIN=local
WORKDIR /source
COPY --from=sources /sources/bbb/bbb-graphql-middleware/ ./
RUN --mount=type=cache,id=bbb-go-mod,target=/go/pkg/mod --mount=type=cache,id=bbb-go-build,target=/root/.cache/go-build \
    go mod download && CGO_ENABLED=0 go build -mod=readonly -trimpath -ldflags="-s -w" -o /out/bbb-graphql-middleware ./cmd/bbb-graphql-middleware/main.go

FROM ubuntu:24.04@sha256:fe81560990fae86dc4743202b2cb4d3df98d0bcf949173cecef15ce1127ef852 AS minimal-runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tini \
    && rm -rf /var/lib/apt/lists/* && groupadd -g 10001 bbb && useradd -u 10001 -g bbb -M -d /tmp bbb
ENV HOME=/tmp LANG=C.UTF-8
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--"]

FROM minimal-runtime AS assembled-runtime
COPY --from=build-middleware /out/bbb-graphql-middleware /usr/local/bin/bbb-graphql-middleware
COPY --from=sources /sources/bbb/bbb-graphql-middleware/config/config.yml /usr/share/bbb-graphql-middleware/config.yml
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh graphql-middleware
RUN mkdir -p /rootfs/tmp && chmod 1777 /rootfs/tmp
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "graphql-middleware"]
CMD ["bbb-graphql-middleware"]

# Both Go binaries and tini-static are self-contained.
FROM scratch AS graphql-middleware
COPY --from=assembled-runtime /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=assembled-runtime /etc/passwd /etc/passwd
COPY --from=assembled-runtime /etc/group /etc/group
COPY --from=assembled-runtime /rootfs/ /
COPY --from=assembled-runtime /usr/local/bin/bbb-runtime /usr/local/bin/bbb-runtime
COPY --from=assembled-runtime /usr/share/bbb-container/ /usr/share/bbb-container/
COPY --from=assembled-runtime /usr/local/bin/bbb-graphql-middleware /usr/local/bin/bbb-graphql-middleware
COPY --from=assembled-runtime /usr/share/bbb-graphql-middleware/ /usr/share/bbb-graphql-middleware/
COPY --from=assembled-runtime /etc/bigbluebutton/ /etc/bigbluebutton/
COPY --from=assembled-runtime /usr/bin/tini-static /usr/bin/tini
ENV HOME=/tmp LANG=C.UTF-8
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "graphql-middleware"]
CMD ["bbb-graphql-middleware"]

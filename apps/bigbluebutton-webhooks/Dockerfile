# syntax=docker/dockerfile:1.7
# Source-built BigBlueButton webhooks; version and target are selected by Bake.
ARG VERSION
# renovate: datasource=github-tags depName=bigbluebutton/bbb-webhooks
ARG WEBHOOKS_VERSION=v3.7.0

FROM golang:1.27-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -ldflags="-s -w" -o /out/bbb-runtime .

FROM ubuntu:24.04@sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac AS addon-sources
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*
ARG WEBHOOKS_VERSION
ADD https://codeload.github.com/bigbluebutton/bbb-webhooks/tar.gz/refs/tags/${WEBHOOKS_VERSION} /tmp/webhooks.tar
RUN mkdir -p /sources/webhooks && tar --no-same-owner -xf /tmp/webhooks.tar --strip-components=1 -C /sources/webhooks && rm /tmp/webhooks.tar

FROM node:24-bookworm-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5 AS build-webhooks
WORKDIR /app
COPY --from=addon-sources /sources/webhooks/ ./
RUN --mount=type=cache,id=bbb-npm-addons,target=/root/.npm npm ci --omit=dev --no-audit --no-fund \
    && cp config/default.example.yml config/default.yml

# Runtime needs Node itself; npm, Yarn and development headers stay in builders.
FROM node:24-bookworm-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5 AS node-binary

FROM debian:bookworm-slim@sha256:7ff6ba31527b9e336162fa061746c366c4b2032f4d2cff2078bea43f78efe3f8 AS webhooks
COPY --from=node-binary /usr/local/bin/node /usr/local/bin/node
RUN apt-get update && apt-get install -y --no-install-recommends tini ca-certificates libstdc++6 libatomic1 && rm -rf /var/lib/apt/lists/* \
    && groupadd -g 1000 node && useradd -u 1000 -g node -m -d /home/node node
WORKDIR /app
COPY --from=build-webhooks /app/ ./
ENV NODE_ENV=production
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --from=node-binary /usr/local/LICENSE /usr/local/share/licenses/node/LICENSE
USER node
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "webhooks"]
CMD ["node", "app.js"]

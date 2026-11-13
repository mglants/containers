# syntax=docker/dockerfile:1.7
# Source-built BigBlueButton frontend; version and target are selected by Bake.
ARG VERSION
# renovate: datasource=github-tags depName=bigbluebutton/bbb-playback
ARG PLAYBACK_VERSION=v5.4.8

FROM ubuntu:24.04@sha256:24737cef17f4e22b1977df7d9a05e77c4e28b7da5b8911f8d8ccda136bf3573c AS bbb-source
ARG VERSION
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*
ADD https://codeload.github.com/bigbluebutton/bigbluebutton/tar.gz/refs/tags/v${VERSION} /tmp/bbb.tar
RUN mkdir -p /sources/bbb && tar --no-same-owner -xf /tmp/bbb.tar --strip-components=1 -C /sources/bbb && rm /tmp/bbb.tar

FROM bbb-source AS sources
ARG PLAYBACK_VERSION
ADD https://codeload.github.com/bigbluebutton/bbb-playback/tar.gz/refs/tags/${PLAYBACK_VERSION} /tmp/playback.tar
RUN mkdir -p /sources/playback && tar --no-same-owner -xf /tmp/playback.tar --strip-components=1 -C /sources/playback && rm /tmp/playback.tar

FROM sources AS staged-nginx
COPY build/stage-nginx.py /build/stage-nginx.py
RUN python3 /build/stage-nginx.py

FROM golang:1.27-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -ldflags="-s -w" -o /out/bbb-runtime .

FROM node:26-bookworm-slim@sha256:bc0a2afd3b74ce12222fc8bab99271e66257f021078f5f4a5d687a77d280c1d9 AS node-builder
ARG VERSION
ENV NODE_OPTIONS=--max-old-space-size=6144
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git python3 make g++ pkg-config && rm -rf /var/lib/apt/lists/*
COPY --from=sources /sources/bbb/ /sources/bbb/
COPY build/build-node.sh /build/build-node.sh
COPY --from=sources /sources/playback/ /sources/playback/

FROM node-builder AS build-frontend
COPY build/patch-client.py /build/patch-client.py
RUN python3 /build/patch-client.py
RUN --mount=type=cache,id=bbb-npm-frontend,target=/root/.npm,sharing=locked bash /build/build-node.sh frontend

# Browser source maps are debugging artifacts; keep shipped application assets.
RUN find /out -type f \( -name "*.map" -o -name "*.map.gz" -o -name "*.map.br" \) -delete

FROM node-builder AS build-playback
ARG PLAYBACK_VERSION
# Inline Python content is supplied by the Dockerfile, not the build context.
# droast ignore=DF077
COPY <<'PYCODE' /tmp/bbb-build-3.py
from pathlib import Path
p = Path('/sources/playback/src/config.js')
s = p.read_text()
assert s.count("url: 'HOST'") == 1
p.write_text(s.replace("url: 'HOST'", "url: '/playback/presentation/2.3'"))
PYCODE
RUN python3 /tmp/bbb-build-3.py && rm /tmp/bbb-build-3.py
# droast ignore=DF077
COPY <<'PYCODE' /tmp/bbb-build-playback.py
import os, subprocess
from pathlib import Path
version = os.environ['PLAYBACK_VERSION'].removeprefix('v')
subprocess.run(['npm', 'run', 'build'], check=True, env={**os.environ, 'REACT_APP_BBB_PLAYBACK_BUILD': version})
Path('build/bbb-playback-version').write_text(version + '\n')
PYCODE
RUN --mount=type=cache,id=bbb-npm-playback,target=/root/.npm,sharing=locked \
    cd /sources/playback && npm ci --no-audit --no-fund \
    && python3 /tmp/bbb-build-playback.py && rm /tmp/bbb-build-playback.py

# Browser source maps are debugging artifacts; keep shipped application assets.
RUN find /sources/playback/build -type f \( -name "*.map" -o -name "*.map.gz" -o -name "*.map.br" \) -delete

FROM nginx:1.31-alpine@sha256:a3554c0d0b48177b1413c26e01f695334428020d9df01243fbae3e11cfd17417 AS frontend
COPY --from=build-frontend /out/client/ /usr/share/bigbluebutton/html5-client/
COPY --from=build-playback /sources/playback/build/ /var/bigbluebutton/playback/presentation/2.3/
COPY --from=build-frontend /out/dashboard/ /var/bigbluebutton/learning-dashboard/
COPY --from=sources /sources/bbb/bigbluebutton-config/assets/ /var/www/bigbluebutton-default/assets/
RUN mkdir -p /var/cache/nginx && chown -R nginx:nginx /var/cache/nginx
EXPOSE 8080
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN rm /etc/nginx/nginx.conf && /usr/share/bbb-container/install-config.sh frontend
COPY --from=staged-nginx /out/nginx/ /usr/share/bbb-container/nginx/
USER nginx
ENTRYPOINT ["/usr/local/bin/bbb-runtime", "frontend"]
CMD ["nginx", "-g", "daemon off;"]

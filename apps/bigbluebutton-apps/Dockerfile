# syntax=docker/dockerfile:1.7
# Source-built BigBlueButton apps; version and target are selected by Bake.
ARG VERSION

FROM ubuntu:24.04@sha256:fe81560990fae86dc4743202b2cb4d3df98d0bcf949173cecef15ce1127ef852 AS sources
ARG VERSION
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*
ADD https://codeload.github.com/bigbluebutton/bigbluebutton/tar.gz/refs/tags/v${VERSION} /tmp/bbb.tar
RUN mkdir -p /sources/bbb && tar --no-same-owner -xf /tmp/bbb.tar --strip-components=1 -C /sources/bbb && rm /tmp/bbb.tar

FROM sources AS client-settings
# Inline Python content is supplied by the Dockerfile, not the build context.
# droast ignore=DF077
COPY <<'PY' /tmp/bbb-build-1.py
import os
from pathlib import Path
version = os.environ['VERSION']
settings = Path('/sources/bbb/bigbluebutton-html5/private/config/settings.yml').read_text()
values = {'HTML5_FULL_BBB_VERSION': version,
          'HTML5_CLIENT_VERSION': version}
for marker, value in values.items():
    assert marker in settings, f'Missing expected client setting: {marker}'
    settings = settings.replace(marker, value)
Path('/client-settings.yml').write_text(settings)
Path('/bigbluebutton-release').write_text('BIGBLUEBUTTON_RELEASE=' + version + '\n')
PY
RUN python3 /tmp/bbb-build-1.py && rm /tmp/bbb-build-1.py

FROM golang:1.27-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -ldflags="-s -w" -o /out/bbb-runtime .

FROM bigbluebutton/bbb-build:v4.0.x-release--2026-10-08-103605@sha256:47672819d9c4a183f9252d8fca3cb1a64eb29739d810692936093734019f8c76 AS jvm-common
ARG VERSION
USER root
COPY --from=sources /sources/bbb /source
COPY build/build-jvm.sh /build/build-jvm.sh
RUN bash /build/build-jvm.sh common

FROM jvm-common AS build-apps
RUN bash /build/build-jvm.sh apps

FROM eclipse-temurin:21-jre-noble@sha256:cd35bb33859641c5374415826a5951845104093eaa401cbe9d3d9023444f1c13 AS java-runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tini \
    && rm -rf /var/lib/apt/lists/* && groupadd -g 10001 bbb && useradd -u 10001 -g bbb -M -d /tmp bbb \
    && mkdir -p /etc/bigbluebutton /var/bigbluebutton /var/log/bigbluebutton \
    && chown 10001:10001 /etc/bigbluebutton /var/bigbluebutton /var/log/bigbluebutton
ENV HOME=/tmp LANG=C.UTF-8
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--"]

FROM java-runtime AS apps
WORKDIR /usr/share/bbb-apps-akka
COPY --from=build-apps /out/ ./
COPY --from=sources /sources/bbb/akka-bbb-apps/src/universal/conf/application.conf /etc/bigbluebutton/bbb-apps-akka.conf
COPY --from=client-settings /client-settings.yml /usr/share/bigbluebutton/html5-client/private/config/settings.yml
COPY runtime/logback.xml ./conf/logback.xml
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh apps
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "apps"]
CMD ["bin/bbb-apps-akka"]

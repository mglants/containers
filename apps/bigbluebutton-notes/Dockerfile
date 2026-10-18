# syntax=docker/dockerfile:1.7
# Shared source builds; per-image Bake definitions select targets and pin BBB.
ARG VERSION
# renovate: datasource=github-tags depName=bigbluebutton/bbb-webrtc-sfu
ARG SIGNALING_VERSION=v2.25.0-beta.2
# renovate: datasource=github-tags depName=bigbluebutton/bbb-webrtc-recorder
ARG RECORDER_VERSION=v0.14.0
# renovate: datasource=github-tags depName=bigbluebutton/bbb-playback
ARG PLAYBACK_VERSION=v5.4.7
# renovate: datasource=github-tags depName=bigbluebutton/bbb-webhooks
ARG WEBHOOKS_VERSION=v3.7.0
# renovate: datasource=github-tags depName=bigbluebutton/bbb-presentation-video
ARG PRESENTATION_VIDEO_VERSION=5.1.0-rc.4
# renovate: datasource=github-tags depName=bigbluebutton/bbb-livekit-stt
ARG STT_VERSION=v0.3.0
# renovate: datasource=github-tags depName=livekit/sip
ARG SIP_VERSION=v1.2.0
# renovate: datasource=github-tags depName=bigbluebutton/perfect-freehand-python
ARG PERFECT_FREEHAND_VERSION=1.2.0
# renovate: datasource=github-tags depName=bigbluebutton/bbb-plugin-live-transcription
ARG TRANSCRIPTION_PLUGIN_VERSION=v0.1.2
# renovate: datasource=github-tags depName=Matroska-Org/foundation-source versioning=regex:^mkclean-(?<major>\d+)\.(?<minor>\d+)\.(?<patch>\d+)$
ARG MKCLEAN_VERSION=mkclean-0.8.10
# renovate: datasource=github-releases depName=hasura/graphql-engine
ARG HASURA_VERSION=v2.49.3
FROM ubuntu:24.04@sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac AS bbb-source
ARG VERSION
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*
ADD https://codeload.github.com/bigbluebutton/bigbluebutton/tar.gz/refs/tags/v${VERSION} /tmp/bbb.tar
RUN mkdir -p /sources/bbb && tar --no-same-owner -xf /tmp/bbb.tar --strip-components=1 -C /sources/bbb && rm /tmp/bbb.tar

# Companion revisions are coordinated with BBB; keep reviewed archives pinned.
FROM bbb-source AS sources
ARG SIGNALING_VERSION
ARG RECORDER_VERSION
ARG PLAYBACK_VERSION
ARG MKCLEAN_VERSION
ARG HASURA_VERSION
ADD https://codeload.github.com/bigbluebutton/bbb-webrtc-sfu/tar.gz/refs/tags/${SIGNALING_VERSION} /tmp/signaling.tar
ADD https://codeload.github.com/bigbluebutton/bbb-webrtc-recorder/tar.gz/refs/tags/${RECORDER_VERSION} /tmp/recorder.tar
ADD https://codeload.github.com/bigbluebutton/bbb-playback/tar.gz/refs/tags/${PLAYBACK_VERSION} /tmp/playback.tar
ADD https://downloads.sourceforge.net/project/matroska/mkclean/${MKCLEAN_VERSION}.tar.bz2 /tmp/mkclean.tar
RUN for source in signaling recorder playback mkclean; do \
      mkdir -p /sources/${source} \
      && tar --no-same-owner -xf /tmp/${source}.tar --strip-components=1 -C /sources/${source} \
      && rm /tmp/${source}.tar || exit 1; \
    done
ADD https://github.com/hasura/graphql-engine/releases/download/${HASURA_VERSION}/cli-hasura-linux-amd64 /tools/hasura
RUN chmod 755 /tools/hasura

FROM sources AS staged-nginx
COPY build/stage-nginx.py /build/stage-nginx.py
RUN python3 /build/stage-nginx.py

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

# A static configuration adapter keeps Python/template tools out of runtime images.
FROM golang:1.27-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -o /out/bbb-runtime .

FROM sources AS runtime-defaults
RUN apt-get update && apt-get install -y --no-install-recommends python3-yaml && rm -rf /var/lib/apt/lists/*
# Inline Python content is supplied by the Dockerfile, not the build context.
# droast ignore=DF077
COPY <<'PYCODE' /tmp/bbb-build-2.py
import json
from pathlib import Path
import yaml
out=Path('/runtime-defaults');out.mkdir()
config=yaml.safe_load(Path('/sources/recorder/config/bbb-webrtc-recorder.yml').read_text())
for name in ('dirFileMode','fileMode'):
    config['recorder'][name]=format(config['recorder'][name], '04o')
(out/'recorder.json').write_text(json.dumps(config))
(out/'annotations.json').write_text(Path('/sources/bbb/bbb-export-annotations/config/settings.json').read_text())
PYCODE
RUN python3 /tmp/bbb-build-2.py && rm /tmp/bbb-build-2.py

FROM node:24-bookworm-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5 AS node-builder
ARG VERSION
ENV NODE_OPTIONS=--max-old-space-size=6144
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git python3 make g++ pkg-config && rm -rf /var/lib/apt/lists/*
COPY --from=sources /sources /sources
COPY build/build-node.sh /build/build-node.sh
FROM node-builder AS build-frontend
COPY build/patch-client.py /build/patch-client.py
RUN python3 /build/patch-client.py
RUN --mount=type=cache,id=bbb-npm-frontend,target=/root/.npm,sharing=locked bash /build/build-node.sh frontend
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
FROM node-builder AS build-actions
COPY build/patch-actions.py /build/patch-actions.py
RUN cd /sources/bbb/bbb-graphql-actions && python3 /build/patch-actions.py
RUN --mount=type=cache,id=bbb-npm,target=/root/.npm,sharing=locked bash /build/build-node.sh graphql-actions
FROM node-builder AS build-notes
RUN --mount=type=cache,id=bbb-npm,target=/root/.npm,sharing=locked bash /build/build-node.sh notes
FROM node-builder AS build-annotations
RUN --mount=type=cache,id=bbb-npm,target=/root/.npm,sharing=locked bash /build/build-node.sh annotations
FROM node-builder AS build-signaling
RUN apt-get update && apt-get install -y --no-install-recommends python3-pip python3-venv \
    && rm -rf /var/lib/apt/lists/*
RUN --mount=type=cache,id=bbb-npm,target=/root/.npm,sharing=locked bash /build/build-node.sh signaling
FROM build-signaling AS staged-signaling
RUN rm -rf /out/.git /out/node_modules/mediasoup/worker/out/Release/subprojects \
    /out/node_modules/mediasoup/worker/out/Release/mediasoup-worker.p \
    /out/node_modules/mediasoup/worker/out/Release/deps

# Builder selected by upstream BBB 4.0 CI; never used as a runtime image.
FROM bigbluebutton/bbb-build:v4.0.x-release--2026-10-08-103605@sha256:47672819d9c4a183f9252d8fca3cb1a64eb29739d810692936093734019f8c76 AS jvm-common
ARG VERSION
USER root
COPY --from=sources /sources/bbb /source
COPY build/build-jvm.sh /build/build-jvm.sh
RUN bash /build/build-jvm.sh common
FROM jvm-common AS build-web
RUN bash /build/build-jvm.sh web
FROM jvm-common AS build-apps
RUN bash /build/build-jvm.sh apps

FROM golang:1.27-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS build-middleware
ENV GOTOOLCHAIN=local
WORKDIR /source
COPY --from=sources /sources/bbb/bbb-graphql-middleware/ ./
RUN --mount=type=cache,id=bbb-go-mod,target=/go/pkg/mod --mount=type=cache,id=bbb-go-build,target=/root/.cache/go-build \
    go mod download && CGO_ENABLED=0 go build -mod=readonly -trimpath -o /out/bbb-graphql-middleware ./cmd/bbb-graphql-middleware/main.go

FROM ubuntu:24.04@sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac AS minimal-runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tini \
    && rm -rf /var/lib/apt/lists/* && groupadd -g 10001 bbb && useradd -u 10001 -g bbb -M -d /tmp bbb
ENV HOME=/tmp LANG=C.UTF-8
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--"]
FROM minimal-runtime AS graphql-middleware
COPY --from=build-middleware /out/bbb-graphql-middleware /usr/local/bin/bbb-graphql-middleware
COPY --from=sources /sources/bbb/bbb-graphql-middleware/config/config.yml /usr/share/bbb-graphql-middleware/config.yml
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh graphql-middleware
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "graphql-middleware"]
CMD ["bbb-graphql-middleware"]

FROM node:24-bookworm-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5 AS node-runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tini \
    && rm -rf /var/lib/apt/lists/* && groupadd -g 10001 bbb && useradd -u 10001 -g bbb -M -d /tmp bbb
ENV NODE_ENV=production HOME=/tmp LANG=C.UTF-8
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--"]
FROM node-runtime AS graphql-actions
WORKDIR /usr/local/bigbluebutton/bbb-graphql-actions
COPY --from=build-actions /out/ ./
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "graphql-actions"]
CMD ["node", "bbb-graphql-actions.js"]
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
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh notes
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "notes"]
CMD ["node", "bbb-shared-notes-server.js"]
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
FROM node-runtime AS signaling
WORKDIR /usr/local/bigbluebutton/bbb-webrtc-sfu
COPY --from=staged-signaling /out/ ./
ENV NODE_CONFIG_DIR=/usr/local/bigbluebutton/bbb-webrtc-sfu/config:/etc/bigbluebutton/bbb-webrtc-sfu
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh signaling
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "signaling"]
CMD ["node", "server.js"]

FROM eclipse-temurin:21-jre-noble@sha256:35501d3f93173b420ac9001658eb68157e813316b865087e345c187fca422c4c AS java-runtime
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
FROM java-runtime AS web
USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
    libreoffice-impress libreoffice-writer imagemagick ghostscript poppler-utils zip unzip file potrace \
    fonts-crosextra-carlito fonts-crosextra-caladea fonts-noto && rm -rf /var/lib/apt/lists/*
COPY --from=build-web /out/web/ /usr/share/bbb-web/
COPY --from=build-web /out/prescheck/ /usr/share/prescheck/
COPY --from=sources /sources/bbb/bigbluebutton-config/slides/ /usr/share/bigbluebutton/blank/
COPY --chmod=755 runtime/run-command.sh /usr/share/bbb-web/run-in-systemd.sh
COPY --chmod=755 runtime/convert-office.sh /usr/share/bbb-libreoffice-conversion/convert.sh
COPY runtime/logback.xml /etc/bigbluebutton/logback.xml
COPY --from=client-settings /bigbluebutton-release /etc/bigbluebutton/bigbluebutton-release
RUN chmod 755 /usr/share/prescheck/prescheck.sh \
    && sed -i '/pattern="PDF"/s/rights="none"/rights="write"/' /etc/ImageMagick-6/policy.xml \
    && sed -i '/potrace/d' /etc/ImageMagick-6/delegates.xml
USER 10001:10001
WORKDIR /usr/share/bbb-web
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh web
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "web"]
CMD ["java", "-Dgrails.env=prod", "-Dserver.address=0.0.0.0", "-Dserver.port=8090", "-Dspring.main.allow-circular-references=true", "-Dlogging.config=/etc/bigbluebutton/logback.xml", "-Dlogback.configurationFile=/etc/bigbluebutton/logback.xml", "-cp", ".", "org.springframework.boot.loader.launch.WarLauncher"]

FROM nginx:1.29-bookworm@sha256:8adbdcb969e2676478ee2c7ad333956f0c8e0e4c5a7463f4611d7a2e7a7ff5dc AS frontend
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

FROM minimal-runtime AS bootstrap
USER root
RUN apt-get update && apt-get install -y --no-install-recommends postgresql-client-16 python3 python3-yaml && rm -rf /var/lib/apt/lists/*
COPY --from=sources /tools/hasura /usr/local/bin/hasura
COPY --from=sources /sources/bbb/bbb-graphql-server/ /usr/share/bbb-graphql-server/
COPY --from=sources /sources/bbb/bbb-shared-notes-server/blocknote_schema.sql /usr/share/bbb-container/blocknote_schema.sql
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "bootstrap"]
CMD ["hasura", "version", "--skip-update-check"]

FROM golang:1.27-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS build-recorder
ARG RECORDER_VERSION
ENV GOTOOLCHAIN=local
WORKDIR /source
COPY --from=sources /sources/recorder/ ./
RUN --mount=type=cache,id=bbb-go-mod,target=/go/pkg/mod --mount=type=cache,id=bbb-go-build,target=/root/.cache/go-build \
    go mod download && CGO_ENABLED=0 go build -mod=readonly -trimpath \
    -ldflags="-X github.com/bigbluebutton/bbb-webrtc-recorder/internal.AppVersion=${RECORDER_VERSION}" \
    -o /out/bbb-webrtc-recorder ./cmd/bbb-webrtc-recorder
FROM minimal-runtime AS recorder
COPY --from=build-recorder /out/bbb-webrtc-recorder /usr/local/bin/bbb-webrtc-recorder
COPY --from=sources /sources/recorder/config/bbb-webrtc-recorder.yml /etc/bbb-webrtc-recorder/bbb-webrtc-recorder.yml
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh recorder
COPY --from=runtime-defaults /runtime-defaults/ /usr/share/bbb-container/defaults/
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "recorder"]
CMD ["bbb-webrtc-recorder"]

FROM ruby:3.2-slim-bookworm@sha256:84140472007d27c7ec641dbc606a47ff9c0aa5f358445947b7350392fa5115e5 AS build-recordings
ARG VERSION
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential libsystemd-dev libffi-dev libxml2-dev libxslt1-dev zlib1g-dev python3 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=sources /sources/bbb/record-and-playback/ /sources/bbb/record-and-playback/
COPY --from=sources /sources/mkclean/ /source-mkclean/
COPY build/stage-recordings.py /build/stage-recordings.py
RUN python3 /build/stage-recordings.py
RUN cd /source-mkclean && sed -i 's/\r//g' mkclean/configure.compiled \
    && ./mkclean/configure.compiled && make -C mkclean \
    && cp release/gcc_linux_x64/mkclean /out/mkclean
WORKDIR /out/core
RUN gem install bundler -v 2.4.20 --no-document \
    && bundle config set --local deployment true && bundle config set --local without test \
    && bundle install && bundle clean \
    && rm -rf vendor/bundle/ruby/*/cache && find vendor/bundle -name '*.o' -delete

FROM ruby:3.2-slim-bookworm@sha256:84140472007d27c7ec641dbc606a47ff9c0aa5f358445947b7350392fa5115e5 AS recordings-base
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tini libsystemd0 libffi8 ffmpeg imagemagick poppler-utils rsync \
    python3 python3-attr python3-cairo python3-gi python3-gi-cairo python3-icu python3-lxml \
    gir1.2-pango-1.0 gir1.2-poppler-0.18 fonts-croscore fonts-dejavu \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -g 10001 bbb && useradd -u 10001 -g bbb -M -d /tmp bbb \
    && mkdir -p /var/log/bigbluebutton /var/bigbluebutton /var/lib/bbb-webrtc-recorder \
    && chown -R 10001:10001 /var/log/bigbluebutton /var/bigbluebutton /var/lib/bbb-webrtc-recorder \
    && gem install bundler -v 2.4.20 --no-document
RUN apt-get update && apt-get install -y --no-install-recommends procps && rm -rf /var/lib/apt/lists/*
COPY --from=build-recordings /out/core/ /usr/local/bigbluebutton/core/
COPY --from=build-recordings /out/mkclean /usr/local/bin/mkclean
COPY build/patch-recording-redis.py /build/patch-recording-redis.py
RUN python3 /build/patch-recording-redis.py
ENV HOME=/tmp LANG=C.UTF-8 RECORDING_FORMATS=presentation QUEUE=rap:archive,rap:publish,rap:process,rap:sanity,rap:captions,rap:events \
    BUNDLE_PATH=/usr/local/bigbluebutton/core/vendor/bundle BUNDLE_WITHOUT=test BUNDLE_FROZEN=true
WORKDIR /usr/local/bigbluebutton/core
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh recordings
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "recordings"]
CMD ["bundle", "exec", "rake", "resque:work"]

# Optional feature images; recordings-base is the internal processing base.
FROM bbb-source AS addon-sources
ARG WEBHOOKS_VERSION
ARG PRESENTATION_VIDEO_VERSION
ARG STT_VERSION
ARG SIP_VERSION
ARG PERFECT_FREEHAND_VERSION
ARG TRANSCRIPTION_PLUGIN_VERSION
ADD https://codeload.github.com/bigbluebutton/bbb-webhooks/tar.gz/refs/tags/${WEBHOOKS_VERSION} /tmp/webhooks.tar
ADD https://codeload.github.com/bigbluebutton/bbb-presentation-video/tar.gz/refs/tags/${PRESENTATION_VIDEO_VERSION} /tmp/presentation-video.tar
ADD https://codeload.github.com/bigbluebutton/bbb-livekit-stt/tar.gz/refs/tags/${STT_VERSION} /tmp/stt.tar
ADD https://codeload.github.com/livekit/sip/tar.gz/refs/tags/${SIP_VERSION} /tmp/sip.tar
ADD https://codeload.github.com/bigbluebutton/perfect-freehand-python/tar.gz/refs/tags/${PERFECT_FREEHAND_VERSION} /tmp/perfect-freehand.tar
ADD https://codeload.github.com/bigbluebutton/bbb-plugin-live-transcription/tar.gz/refs/tags/${TRANSCRIPTION_PLUGIN_VERSION} /tmp/transcription-plugin.tar
RUN for source in webhooks presentation-video stt sip perfect-freehand transcription-plugin; do \
      mkdir -p /sources/${source} \
      && tar --no-same-owner -xf /tmp/${source}.tar --strip-components=1 -C /sources/${source} \
      && rm /tmp/${source}.tar || exit 1; \
    done

FROM node:24-bookworm-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5 AS build-webhooks
WORKDIR /app
COPY --from=addon-sources /sources/webhooks/ ./
RUN --mount=type=cache,id=bbb-npm-addons,target=/root/.npm npm ci --omit=dev --no-audit --no-fund \
    && cp config/default.example.yml config/default.yml
FROM node:24-bookworm-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5 AS webhooks
RUN apt-get update && apt-get install -y --no-install-recommends tini ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build-webhooks /app/ ./
ENV NODE_ENV=production
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
USER node
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "webhooks"]
CMD ["node", "app.js"]

FROM node:24-bookworm-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5 AS build-plugins
WORKDIR /app
COPY --from=addon-sources /sources/transcription-plugin/ ./
RUN --mount=type=cache,id=bbb-npm-addons,target=/root/.npm npm ci --no-audit --no-fund && npm run build-bundle
FROM nginx:1.29-bookworm@sha256:8adbdcb969e2676478ee2c7ad333956f0c8e0e4c5a7463f4611d7a2e7a7ff5dc AS plugins
COPY --from=build-plugins /app/dist/ /usr/share/nginx/html/live-transcription/
COPY build/plugins.conf /etc/nginx/nginx.conf
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh plugins
USER nginx
ENTRYPOINT ["/usr/local/bin/bbb-runtime", "plugins"]
CMD ["nginx", "-g", "daemon off;"]

FROM golang:1.27-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS build-sip
ENV GOTOOLCHAIN=local
RUN apt-get update && apt-get install -y --no-install-recommends pkg-config libopus-dev libopusfile-dev libsoxr-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=addon-sources /sources/sip/ ./
COPY --from=addon-sources /sources/bbb/build/packages-template/bbb-livekit/no-pin-sub_sip.patch /build/no-pin-sub_sip.patch
RUN git apply /build/no-pin-sub_sip.patch
RUN --mount=type=cache,id=bbb-go-mod,target=/go/pkg/mod --mount=type=cache,id=bbb-go-build,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -mod=readonly -trimpath -o /out/livekit-sip ./cmd/livekit-sip
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

FROM python:3.12-slim-bookworm@sha256:34386ef0cb081344d7ec1c103ba398e6e9f64e9ab3a1509accc92a4e24a07258 AS stt
RUN apt-get update && apt-get install -y --no-install-recommends tini ca-certificates libglib2.0-0 libgomp1 && rm -rf /var/lib/apt/lists/* \
    && pip install --no-cache-dir uv==0.10.4
WORKDIR /app
COPY --from=addon-sources /sources/stt/ ./
RUN UV_PYTHON_DOWNLOADS=never uv sync --python /usr/local/bin/python3 --frozen --no-dev --no-cache
ENV PATH=/app/.venv/bin:$PATH HOME=/tmp
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "stt"]
CMD ["python3", "main.py", "start"]

FROM recordings-base AS recordings
ARG PRESENTATION_VIDEO_VERSION
USER root
RUN apt-get update && apt-get install -y --no-install-recommends python3-venv python3-setuptools python3-setuptools-scm python3-packaging python3-sortedcollections gir1.2-gtk-3.0 && rm -rf /var/lib/apt/lists/*
RUN apt-get update && apt-get install -y --no-install-recommends python3-wheel && rm -rf /var/lib/apt/lists/*
COPY --from=addon-sources /sources/presentation-video/ /source/presentation-video/
COPY --from=addon-sources /sources/perfect-freehand/ /source/perfect-freehand/
RUN python3 -m venv --system-site-packages /opt/renderer \
    && SETUPTOOLS_SCM_PRETEND_VERSION=${PRESENTATION_VIDEO_VERSION} /opt/renderer/bin/pip install --no-deps --no-build-isolation /source/perfect-freehand /source/presentation-video \
    && rm -rf /source
COPY --from=addon-sources /sources/bbb/record-and-playback/video/playback/video/ /usr/local/bigbluebutton/core/playback/video/
# Inline Python content is supplied by the Dockerfile, not the build context.
# droast ignore=DF077
COPY <<'PYCODE' /tmp/bbb-build-4.py
from pathlib import Path
for name in ('process', 'publish'):
    p = Path('/usr/local/bigbluebutton/core/scripts') / name / 'video.rb'
    text = p.read_text()
    assert 'Logger.new(log_file)' in text or 'Logger.new(logfile)' in text
    p.write_text(text.replace('Logger.new(log_file)', 'Logger.new($stdout)').replace('Logger.new(logfile)', 'Logger.new($stdout)'))
PYCODE
RUN python3 /tmp/bbb-build-4.py && rm /tmp/bbb-build-4.py
ENV PATH=/opt/renderer/bin:$PATH RECORDING_FORMATS=presentation,video
USER 10001:10001

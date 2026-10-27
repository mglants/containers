# syntax=docker/dockerfile:1.7
# Source-built BigBlueButton recorder; version and target are selected by Bake.
ARG VERSION
# renovate: datasource=github-tags depName=bigbluebutton/bbb-webrtc-recorder
ARG RECORDER_VERSION=v0.18.0

FROM ubuntu:24.04@sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac AS sources
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*
ARG RECORDER_VERSION
ADD https://codeload.github.com/bigbluebutton/bbb-webrtc-recorder/tar.gz/refs/tags/${RECORDER_VERSION} /tmp/recorder.tar
RUN mkdir -p /sources/recorder && tar --no-same-owner -xf /tmp/recorder.tar --strip-components=1 -C /sources/recorder && rm /tmp/recorder.tar

FROM golang:1.27-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -ldflags="-s -w" -o /out/bbb-runtime .

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
PYCODE
RUN python3 /tmp/bbb-build-2.py && rm /tmp/bbb-build-2.py

FROM ubuntu:24.04@sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac AS minimal-runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tini \
    && rm -rf /var/lib/apt/lists/* && groupadd -g 10001 bbb && useradd -u 10001 -g bbb -M -d /tmp bbb
ENV HOME=/tmp LANG=C.UTF-8
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--"]

FROM golang:1.27-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5 AS build-recorder
ARG RECORDER_VERSION
ENV GOTOOLCHAIN=local
WORKDIR /source
COPY --from=sources /sources/recorder/ ./
RUN --mount=type=cache,id=bbb-go-mod,target=/go/pkg/mod --mount=type=cache,id=bbb-go-build,target=/root/.cache/go-build \
    go mod download && CGO_ENABLED=0 go build -mod=readonly -trimpath \
    -ldflags="-s -w -X github.com/bigbluebutton/bbb-webrtc-recorder/internal.AppVersion=${RECORDER_VERSION}" \
    -o /out/bbb-webrtc-recorder ./cmd/bbb-webrtc-recorder

FROM minimal-runtime AS assembled-runtime
COPY --from=build-recorder /out/bbb-webrtc-recorder /usr/local/bin/bbb-webrtc-recorder
COPY --from=sources /sources/recorder/config/bbb-webrtc-recorder.yml /etc/bbb-webrtc-recorder/bbb-webrtc-recorder.yml
USER root
COPY --from=build-runtime /out/bbb-runtime /usr/local/bin/bbb-runtime
COPY --chmod=755 runtime/install-config.sh /usr/share/bbb-container/install-config.sh
RUN /usr/share/bbb-container/install-config.sh recorder
RUN mkdir -p /rootfs/tmp && chmod 1777 /rootfs/tmp
COPY --from=runtime-defaults /runtime-defaults/ /usr/share/bbb-container/defaults/
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "recorder"]
CMD ["bbb-webrtc-recorder"]

# Both Go binaries and tini-static are self-contained.
FROM scratch AS recorder
COPY --from=assembled-runtime /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=assembled-runtime /etc/passwd /etc/passwd
COPY --from=assembled-runtime /etc/group /etc/group
COPY --from=assembled-runtime /rootfs/ /
COPY --from=assembled-runtime /usr/local/bin/bbb-runtime /usr/local/bin/bbb-runtime
COPY --from=assembled-runtime /usr/share/bbb-container/ /usr/share/bbb-container/
COPY --from=assembled-runtime /usr/local/bin/bbb-webrtc-recorder /usr/local/bin/bbb-webrtc-recorder
COPY --from=assembled-runtime /etc/bbb-webrtc-recorder/ /etc/bbb-webrtc-recorder/
COPY --from=assembled-runtime /usr/bin/tini-static /usr/bin/tini
ENV HOME=/tmp LANG=C.UTF-8
USER 10001:10001
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/bbb-runtime", "recorder"]
CMD ["bbb-webrtc-recorder"]

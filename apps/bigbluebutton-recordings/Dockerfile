# syntax=docker/dockerfile:1.7
# Source-built BigBlueButton recordings; version and target are selected by Bake.
ARG VERSION
# renovate: datasource=github-tags depName=bigbluebutton/bbb-presentation-video
ARG PRESENTATION_VIDEO_VERSION=5.1.0-rc.4
# renovate: datasource=github-tags depName=bigbluebutton/perfect-freehand-python
ARG PERFECT_FREEHAND_VERSION=1.2.0
# renovate: datasource=github-tags depName=Matroska-Org/foundation-source versioning=regex:^mkclean-(?<major>\d+)\.(?<minor>\d+)\.(?<patch>\d+)$
ARG MKCLEAN_VERSION=mkclean-0.9.0

FROM ubuntu:26.04@sha256:6a06111c3b0a8600feadb270aec771dc239bf407b25625f93f44ab00e11f4d3c AS bbb-source
ARG VERSION
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*
ADD https://codeload.github.com/bigbluebutton/bigbluebutton/tar.gz/refs/tags/v${VERSION} /tmp/bbb.tar
RUN mkdir -p /sources/bbb && tar --no-same-owner -xf /tmp/bbb.tar --strip-components=1 -C /sources/bbb && rm /tmp/bbb.tar

FROM bbb-source AS sources
ARG MKCLEAN_VERSION
ADD https://codeload.github.com/Matroska-Org/foundation-source/tar.gz/refs/tags/${MKCLEAN_VERSION} /tmp/mkclean.tar
RUN mkdir -p /sources/mkclean && tar --no-same-owner -xf /tmp/mkclean.tar --strip-components=1 -C /sources/mkclean && rm /tmp/mkclean.tar

FROM golang:1.27-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -ldflags="-s -w" -o /out/bbb-runtime .

# BBB 4.0's Gemfile.lock contains gems that require Ruby < 4.0.
FROM ruby:3.4-slim-bookworm@sha256:549e18ddab0acedc09a63be26cc3bdb6c05988bf3ebccfd34fbc90869395097c AS build-recordings
ARG VERSION
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential cmake libsystemd-dev libffi-dev libxml2-dev libxslt1-dev zlib1g-dev python3 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=sources /sources/bbb/record-and-playback/ /sources/bbb/record-and-playback/
COPY --from=sources /sources/mkclean/ /source-mkclean/
COPY build/stage-recordings.py /build/stage-recordings.py
RUN python3 /build/stage-recordings.py
RUN cmake -S /source-mkclean -B /tmp/mkclean-build -DCMAKE_BUILD_TYPE=Release \
    && cmake --build /tmp/mkclean-build --target mkclean \
    && cp /tmp/mkclean-build/mkclean/mkclean /out/mkclean
WORKDIR /out/core

# Match the BUNDLED WITH version in upstream BBB's Gemfile.lock.
RUN gem install bundler -v 2.4.20 --no-document \
    && bundle config set --local deployment true && bundle config set --local without test \
    && bundle install && bundle clean \
    && rm -rf vendor/bundle/ruby/*/cache && find vendor/bundle -name '*.o' -delete

FROM ruby:3.4-slim-bookworm@sha256:549e18ddab0acedc09a63be26cc3bdb6c05988bf3ebccfd34fbc90869395097c AS recordings-base
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

FROM bbb-source AS addon-sources
ARG PERFECT_FREEHAND_VERSION
ADD https://codeload.github.com/bigbluebutton/perfect-freehand-python/tar.gz/refs/tags/${PERFECT_FREEHAND_VERSION} /tmp/perfect-freehand.tar
RUN mkdir -p /sources/perfect-freehand && tar --no-same-owner -xf /tmp/perfect-freehand.tar --strip-components=1 -C /sources/perfect-freehand && rm /tmp/perfect-freehand.tar
ARG PRESENTATION_VIDEO_VERSION
ADD https://codeload.github.com/bigbluebutton/bbb-presentation-video/tar.gz/refs/tags/${PRESENTATION_VIDEO_VERSION} /tmp/presentation-video.tar
RUN mkdir -p /sources/presentation-video && tar --no-same-owner -xf /tmp/presentation-video.tar --strip-components=1 -C /sources/presentation-video && rm /tmp/presentation-video.tar

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
COPY runtime/summary.rb /usr/local/bigbluebutton/core/lib/recordandplayback/summary.rb
# droast ignore=DF077
COPY <<'PYCODE' /tmp/bbb-summary-entrypoints.py
from pathlib import Path
root = Path('/usr/local/bigbluebutton/core/scripts')
for phase in ('process', 'publish'):
    (root / phase / 'summary.rb').write_text("require_relative '../../lib/recordandplayback/summary'\n" + f"BigBlueButton::Summary.run('{phase}', ARGV)\n")
PYCODE
RUN python3 /tmp/bbb-summary-entrypoints.py && rm /tmp/bbb-summary-entrypoints.py
RUN --mount=type=bind,source=runtime/summary_test.rb,target=/tmp/summary_test.rb,readonly \
    bundle exec ruby /tmp/summary_test.rb
ENV PATH=/opt/renderer/bin:$PATH RECORDING_FORMATS=presentation,video
USER 10001:10001

# syntax=docker/dockerfile:1.7
# Source-built BigBlueButton stt; version and target are selected by Bake.
ARG VERSION
# renovate: datasource=github-tags depName=bigbluebutton/bbb-livekit-stt
ARG STT_VERSION=v0.3.0

FROM golang:1.27-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS build-runtime
ENV GOTOOLCHAIN=local GO111MODULE=off CGO_ENABLED=0
WORKDIR /source
COPY runtime/config/ ./
RUN --mount=type=cache,id=bbb-runtime-go-build,target=/root/.cache/go-build \
    go test . && go build -trimpath -o /out/bbb-runtime .

FROM ubuntu:24.04@sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac AS addon-sources
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 bzip2 && rm -rf /var/lib/apt/lists/*
ARG STT_VERSION
ADD https://codeload.github.com/bigbluebutton/bbb-livekit-stt/tar.gz/refs/tags/${STT_VERSION} /tmp/stt.tar
RUN mkdir -p /sources/stt && tar --no-same-owner -xf /tmp/stt.tar --strip-components=1 -C /sources/stt && rm /tmp/stt.tar

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

FROM --platform=$TARGETPLATFORM ghcr.io/onedr0p/alpine:rolling@sha256:3fbc581cb0fe29830376161ae026e2a765dcc11e1747477fe9ebf155720b8638

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG VERSION
ARG CHANNEL
ARG TARGETARCH
LABEL xyz.glants.image.target_platform=$TARGETPLATFORM
LABEL xyz.glants.image.target_architecture=$TARGETARCH
LABEL xyz.glants.image.target_os=$TARGETOS
USER root
RUN apk add tuned --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing/
CMD tuned-adm

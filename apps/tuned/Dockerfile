FROM --platform=$TARGETPLATFORM docker.io/alpine:3.23.2

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
COPY --chown=root:root ./tuned-main.conf /etc/tuned/tuned-main.conf
CMD tuned

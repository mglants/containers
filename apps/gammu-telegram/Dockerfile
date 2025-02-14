# syntax=docker/dockerfile:experimental

# Use the Alpine Linux 3 base image
FROM docker.io/alpine:3.20.6

# Define build arguments
ARG TARGETPLATFORM
ARG TARGETARCH
ARG TARGETOS

# Set the maintainer label
LABEL xyz.glants.image.target_platform=$TARGETPLATFORM
LABEL xyz.glants.image.target_architecture=$TARGETARCH
LABEL xyz.glants.image.target_os=$TARGETOS
LABEL org.opencontainers.image.title="gammu-telegram"
LABEL org.opencontainers.image.source="https://github.com/gammu/gammu"

RUN apk add --update --no-cache python3 py3-requests bash gammu-smsd envsubst && \
    rm -rf /var/cache/apk/*

ADD --chown=nobody:nobody ./scripts /scripts
ADD --chown=nobody:nobody ./entrypoint.sh /entrypoint.sh
RUN mkdir /data
RUN chown nobody:nobody /data

RUN mkdir /app
RUN chown nobody:nobody /app
# Set the user to nobody
USER nobody

ENV GAMMU_DEVICE=/dev/ttyUSB0
ENV GAMMU_CONNECTION=at

ENTRYPOINT ["/entrypoint.sh"]

CMD ["gammu-smsd", "-c",  "/app/gammu.conf"]

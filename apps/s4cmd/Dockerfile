# syntax=docker/dockerfile:experimental

# Use the Alpine Linux 3 base image
FROM docker.io/alpine:3.23.2

# Define build arguments
ARG VERSION
ARG TARGETPLATFORM
ARG TARGETARCH
ARG TARGETOS

# Set the maintainer label
LABEL xyz.glants.image.target_platform=$TARGETPLATFORM
LABEL xyz.glants.image.target_architecture=$TARGETARCH
LABEL xyz.glants.image.target_os=$TARGETOS
LABEL org.opencontainers.image.title="s4cmd"
LABEL org.opencontainers.image.source="https://github.com/bloomreach/s4cmd"

RUN apk --update --no-cache add git python3 py3-pip bash curl wget tar zip && \
    pip install --break-system-packages git+https://github.com/bloomreach/s4cmd.git@${VERSION} && \
    rm -rf /var/cache/apk/*


# Set the user to nobody
USER nobody


ENTRYPOINT ["/usr/bin/s4cmd"]

CMD ["--help"]

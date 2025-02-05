ARG VERSION
FROM docker.io/golang:1.23.6 AS build
ARG VERSION

RUN \
    apt-get update && apt-get install git \
    && git clone --branch "v${VERSION}" "https://github.com/poseidon/matchbox.git" src \
    && apt-get clean
RUN cd src && make build

FROM docker.io/alpine:3.20.0
RUN mkdir /var/lib/matchbox && mkdir /var/lib/matchbox/assets
ARG TARGETPLATFORM
ARG TARGETARCH
ARG TARGETOS
LABEL xyz.glants.image.target_platform=$TARGETPLATFORM
LABEL xyz.glants.image.target_architecture=$TARGETARCH
LABEL xyz.glants.image.target_os=$TARGETOS
LABEL org.opencontainers.image.title="Matchbox"
LABEL org.opencontainers.image.source="https://github.com/poseidon/matchbox"
LABEL org.opencontainers.image.documentation="https://matchbox.psdn.io"
LABEL org.opencontainers.image.vendor="Poseidon Labs"

COPY --from=build /go/src/bin/matchbox /matchbox
EXPOSE 8080
ENTRYPOINT ["/matchbox"]

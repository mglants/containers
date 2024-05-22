FROM docker.io/python:alpine3.19 as build
ARG VERSION
ADD https://github.com/akpw/mktxp/archive/refs/tags/v${VERSION}.tar.gz /
RUN mkdir /mktxp && \
      tar -zxvf /*.tar.gz -C /mktxp && \
      rm /*.tar.gz
RUN pip3 install --target="/install" /mktxp/mktxp-${VERSION}

FROM docker.io/alpine:3.20.0

# Install runtime dependencies
ARG RUN_PKGS="python3"
RUN apk add --update --no-cache ${RUN_PKGS} \
    && rm -rf /var/cache/apk/* /usr/local/share/man/* \
    && mkdir -p /var/lib/kea
ENV PYTHONPATH "${PYTHONPATH}:/usr/local"
ARG TARGETPLATFORM
ARG TARGETARCH
ARG TARGETOS
LABEL xyz.glants.image.target_platform=$TARGETPLATFORM
LABEL xyz.glants.image.target_architecture=$TARGETARCH
LABEL xyz.glants.image.target_os=$TARGETOS
LABEL org.opencontainers.image.title="mktxp"
LABEL org.opencontainers.image.source="github.com/akpw/mktxp"
COPY --from=build /install /usr/local
RUN addgroup -S mktxp && adduser -S mktxp -G mktxp
RUN mkdir /config
RUN chown mktxp:mktxp /config

EXPOSE 49090
USER mktxp
CMD [ "python3", "-u", "/usr/local/bin/mktxp", "--cfg-dir", "/config",  "export"]
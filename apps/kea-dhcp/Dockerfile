# build binary
FROM docker.io/alpine:3.19.1 AS build
ARG VERSION
# Environment
ARG BUILD_PKGS="build-base boost-dev log4cplus-dev autoconf automake bison botan-dev flex libcap-utils libtool mariadb-dev postgresql-dev python3-dev procps curl"
ARG CFLAGS="-O2 -pthread -pipe -fPIC -fPIE -fomit-frame-pointer "
ARG CXXFLAGS="${CFLAGS} "

# dev package install
RUN apk add --no-cache --update ${BUILD_PKGS}
## build
RUN mkdir -p /tmp/build \
  && curl -fSsL "https://ftp.isc.org/isc/kea/${VERSION}/kea-${VERSION}.tar.gz" \
          -o kea.tar.gz \
  && tar xf kea.tar.gz --strip-components=1 -C /tmp/build \
  && rm -f kea.tar.gz
RUN cd /tmp/build \
  && autoreconf -if \
  && ./configure \
        --prefix=/usr/local \
        --sysconfdir=/config \
        --localstatedir=/data \
        --runstatedir=/run \
        --disable-rpath \
        --disable-static \
        --with-boost-include \
        --with-botan-config \
        --with-mysql \
        --with-pgsql \
        --enable-shell \
        --with-log4cplus \
        CFLAGS="${CFLAGS}" CXXFLAGS="${CFLAGS}" LDFLAGS="${LDFLAGS}"
RUN cd /tmp/build \
    && make -j"$(nproc)" \
    && make install DESTDIR=/tmp/root
RUN ldconfig /
# Delete unnecessary directory
RUN rm -rf /tmp/root/var/run /tmp/root/usr/local/share/man/* /tmp/root/usr/local/include /tmp/root/usr/include \
           /tmp/root/usr/share/kea /tmp/root/usr/share/doc/kea /tmp/root/usr/local/lib/kea/hooks/*.la \
           /tmp/root/usr/local/lib/*.la /tmp/root/usr/local/include/* /tmp/root/data/log/
# Patch config files for patching sockets and logs
COPY ./patches /patches
RUN cd  /tmp/root/config/kea && \
    patch < /patches/kea-ctrl-agent.patch && \
    patch < /patches/kea-dhcp-ddns.patch && \
    patch < /patches/kea-dhcp4.patch && \
    patch < /patches/kea-dhcp6.patch
# Intermediate container with runtime dependencies
FROM docker.io/python:alpine3.19 as exporter
RUN mkdir /install
WORKDIR /install
RUN pip3 install --target="/install" kea-exporter

# Intermediate container with runtime dependencies
FROM docker.io/alpine:3.19.1 AS runtime

# Install runtime dependencies
ARG RUN_PKGS="libgcc libstdc++ musl botan-libs mariadb-connector-c socat libpq procps \
         python3 log4cplus"
RUN apk add --update --no-cache ${RUN_PKGS} \
    && rm -rf /var/cache/apk/* /usr/local/share/man/* \
    && mkdir -p /var/lib/kea

# Final container
FROM runtime
ENV PYTHONPATH "${PYTHONPATH}:/usr/local"
ARG TARGETPLATFORM
ARG TARGETARCH
ARG TARGETOS
LABEL xyz.glants.image.target_platform=$TARGETPLATFORM
LABEL xyz.glants.image.target_architecture=$TARGETARCH
LABEL xyz.glants.image.target_os=$TARGETOS
LABEL org.opencontainers.image.title="kea-dhcp"
LABEL org.opencontainers.image.source="https://github.com/isc-projects/kea"
LABEL org.opencontainers.image.documentation="https://kea.readthedocs.io/"
LABEL org.opencontainers.image.vendor="isc-projects"

#Logging
RUN mkdir /var/log/kea
ENV KEA_LOGGER_DESTINATION stdout
ENV KEA_LOCKFILE_DIR /run/kea
# publish port TCP and UDP
EXPOSE 67/UDP 547/TCP 547/UDP 8080 9547

# service running
STOPSIGNAL SIGTERM

# select entrypoint
COPY ./entrypoint.sh /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

# Fetch isc Kea libraries from build image
COPY --from=build /tmp/root/ /
COPY --from=exporter /install /usr/local
RUN ldconfig /
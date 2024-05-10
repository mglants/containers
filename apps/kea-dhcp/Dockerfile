## base image
FROM docker.io/alpine:3.19.1 AS base-image
ARG BUILD_BASE="alpine-sdk clang llvm lld gawk autoconf automake clang-dev libtool"
ARG BUILD_PKGS="${BUILD_BASE}"
ARG CFLAGS="-O2 -pthread -pipe -fPIC -fPIE -fomit-frame-pointer "
ARG CXXFLAGS="${CFLAGS} "
ARG LDFLAGS="-Wl,-O2 -Wl,--as-needed -Wl,-z,relro -Wl,-z,now "
ENV CC="clang" \
    CXX="clang++" \
    AR="llvm-ar" \
    NM="llvm-nm" \
    RANLIB="llvm-ranlib" \
    LD="ld.lld" \
    STRIP="llvm-strip"

# make base image
RUN apk add --no-cache --update ${BUILD_PKGS}

# build binary
FROM base-image AS build
ARG VERSION
# Environment
ARG BUILD_PKGS="boost-dev postgresql-dev log4cplus-dev botan-dev mariadb-dev bison flex perl curl openssl-dev cassandra-cpp-driver-dev"
ARG CFLAGS="-O2 -pthread -pipe -fPIC -fPIE -fomit-frame-pointer "
ARG CXXFLAGS="${CFLAGS} "

# dev package install
COPY --from=log4cplus /tmp/root /
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
        --sysconfdir=/etc \
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
           /tmp/root/usr/local/lib/*.la /tmp/root/usr/local/include/*



# Intermediate container with runtime dependencies
FROM docker.io/python:alpine3.19 as exporter
RUN mkdir /install
WORKDIR /install
RUN pip3 install --target="/install" kea-exporter

# Intermediate container with runtime dependencies
FROM docker.io/alpine:3.19.1 AS runtime

# Install runtime dependencies
ARG RUN_PKGS="libgcc libstdc++ boost-system mariadb-connector-c libpq tzdata procps libatomic tini sqlite-libs \
              libbz2 cassandra-cpp-driver python3 "
COPY --from=log4cplus /tmp/root /
COPY --from=botan /tmp/root /
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

# publish port TCP and UDP
EXPOSE 67/UDP 547/TCP 547/UDP 8080 9547

# service running
STOPSIGNAL SIGTERM

# select entrypoint
WORKDIR /etc/kea
COPY ./entrypoint.sh /usr/local/bin/
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/usr/local/bin/entrypoint.sh"]

# Fetch isc Kea libraries from build image
COPY --from=build /tmp/root/ /
COPY --from=exporter /install /usr/local
USER nobody
RUN ldconfig /
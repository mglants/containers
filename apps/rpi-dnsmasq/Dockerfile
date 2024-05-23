FROM docker.io/alpine:3.20 as rpi
ARG VERSION
COPY ./serials /serials

ADD https://github.com/pftf/RPi4/releases/download/v${VERSION}/RPi4_UEFI_Firmware_v${VERSION}.zip RPi4_UEFI_Firmware.zip

RUN apk add --update --no-cache \
    unzip \
    && mkdir /rpi4 \
    && mv RPi4_UEFI_Firmware.zip /rpi4/RPi4_UEFI_Firmware.zip \
    && cd /rpi4 \
    && ls \
    && unzip RPi4_UEFI_Firmware.zip \
    && rm RPi4_UEFI_Firmware.zip \
    && mkdir /tftp \
    && ls /serials | while read serial; do mkdir /tftp/$serial && cp -r /rpi4/* /tftp/$serial && cp -r /serials/$serial/* /tftp/$serial/; done
FROM quay.io/poseidon/dnsmasq:v0.5.0-31-gdc1adc8
LABEL xyz.glants.image.target_platform=$TARGETPLATFORM
LABEL xyz.glants.image.target_architecture=$TARGETARCH
LABEL xyz.glants.image.target_os=$TARGETOS
LABEL org.opencontainers.image.title="rpi-dnsmasq"
LABEL org.opencontainers.image.source="github.com/poseidon/dnsmasq"
COPY --from=rpi /tftp /var/lib/tftpboot
RUN curl -s -o var/lib/tftpboot/ipxe-arm64.efi http://boot.ipxe.org/ipxe.efi
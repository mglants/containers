FROM docker.io/library/alpine:3.19 as rpi

COPY ./serials /serials

ADD https://github.com/pftf/RPi4/releases/download/v1.37/RPi4_UEFI_Firmware_v1.37.zip RPi4_UEFI_Firmware.zip

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
COPY --from=rpi /tftp /var/lib/tftpboot
#!/usr/bin/env bash

version="$(curl -sX GET 'https://gitlab.alpinelinux.org/alpine/aports/-/raw/master/testing/tuned/APKBUILD' | grep 'pkgver=')"
version="${version#*pkgver=}"
printf "%s" "${version}"

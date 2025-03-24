#!/usr/bin/env bash

version="$(curl -sX GET "https://api.github.com/repos/containernetworking/plugins/releases/latest" | jq --raw-output '.tag_name')"
version="${version#*pkgver=}"
printf "%s" "${version}"

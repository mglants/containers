target "docker-metadata-action" {}

variable "APP" {
  default = "rpi-dnsmasq"
}

variable "VERSION" {
  // renovate: datasource=github-releases depName=pftf/RPi4 versioning=semver-coerced
  default = "v1.52"
}

variable "SOURCE" {
  default = "https://github.com/pftf/RPi4"
}

group "default" {
  targets = ["image-local"]
}

target "image" {
  inherits = ["docker-metadata-action"]
  args = {
    VERSION = trimprefix(VERSION, "v")
  }
  labels = {
    "org.opencontainers.image.source" = SOURCE
  }
}

target "image-local" {
  inherits = ["image"]
  output = ["type=docker"]
  tags = ["${APP}:${VERSION}"]
}

target "image-all" {
  inherits = ["image"]
  platforms = ["linux/arm64", "linux/amd64"]
}

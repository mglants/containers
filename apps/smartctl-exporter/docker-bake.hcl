target "docker-metadata-action" {}

variable "APP" {
  default = "smartctl-exporter"
}

variable "VERSION" {
  // renovate: datasource=github-releases depName=prometheus-community/smartctl_exporter versioning=semver
  default = "v0.14.0"
}

variable "SOURCE" {
  default = "https://github.com/prometheus-community/smartctl_exporter"
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

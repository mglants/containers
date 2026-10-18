target "docker-metadata-action" {}

variable "APP" {
  default = "cni-plugins"
}

variable "VERSION" {
  // renovate: datasource=github-releases depName=containernetworking/plugins versioning=semver
  default = "v1.5.1"
}

variable "SOURCE" {
  default = "https://github.com/containernetworking/plugins"
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
  platforms = ["linux/amd64", "linux/arm64"]
}

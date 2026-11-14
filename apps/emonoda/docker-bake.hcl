target "docker-metadata-action" {}

variable "APP" {
  default = "emonoda"
}

variable "VERSION" {
  // renovate: datasource=github-tags depName=mdevaev/emonoda versioning=semver
  default = "v2.1.40"
}

variable "SOURCE" {
  default = "https://github.com/mdevaev/emonoda"
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

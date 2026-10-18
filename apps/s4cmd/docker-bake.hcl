target "docker-metadata-action" {}

variable "APP" {
  default = "s4cmd"
}

variable "VERSION" {
  // renovate: datasource=github-tags depName=bloomreach/s4cmd versioning=semver
  default = "2.1.0"
}

variable "SOURCE" {
  default = "https://github.com/bloomreach/s4cmd"
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

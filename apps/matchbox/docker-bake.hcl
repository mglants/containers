target "docker-metadata-action" {}

variable "APP" {
  default = "matchbox"
}

variable "VERSION" {
  // renovate: datasource=github-releases depName=poseidon/matchbox versioning=semver
  default = "v0.11.0"
}

variable "SOURCE" {
  default = "https://github.com/poseidon/matchbox"
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
  platforms = ["linux/arm/v7"]
}

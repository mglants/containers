target "docker-metadata-action" {}

variable "APP" {
  default = "tuned"
}

variable "VERSION" {
  // renovate: datasource=github-tags depName=redhat-performance/tuned versioning=semver
  default = "v2.28.0"
}

variable "SOURCE" {
  default = "https://github.com/redhat-performance/tuned"
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

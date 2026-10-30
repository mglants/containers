target "docker-metadata-action" {}

variable "APP" {
  default = "mktxp"
}

variable "VERSION" {
  // renovate: datasource=github-releases depName=akpw/mktxp versioning=semver
  default = "v1.2.20"
}

variable "SOURCE" {
  default = "https://github.com/akpw/mktxp"
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

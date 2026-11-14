target "docker-metadata-action" {}

variable "APP" {
  default = "kea-dhcp"
}

variable "VERSION" {
  // renovate: datasource=github-tags depName=isc-projects/kea versioning=regex:^(?:Kea-)?(?<major>\d+)\.(?<minor>\d+)\.(?<patch>\d+)$
  default = "Kea-2.7.9"
}

variable "SOURCE" {
  default = "https://github.com/isc-projects/kea"
}

group "default" {
  targets = ["image-local"]
}

target "image" {
  inherits = ["docker-metadata-action"]
  args = {
    VERSION = trimprefix(VERSION, "Kea-")
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

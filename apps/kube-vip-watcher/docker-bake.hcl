target "docker-metadata-action" {}

variable "APP" {
  default = "kube-vip-watcher"
}

variable "VERSION" {
  // renovate: datasource=github-tags depName=BBQigniter/kube-vip-watcher versioning=semver-coerced
  default = "v0.11"
}

variable "SOURCE" {
  default = "https://github.com/BBQigniter/kube-vip-watcher"
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

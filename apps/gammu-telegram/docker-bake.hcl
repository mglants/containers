target "docker-metadata-action" {}

variable "APP" {
  default = "gammu-telegram"
}

variable "VERSION" {
  default = "v0.0.3"
}

variable "SOURCE" {
  default = "https://github.com/mglants/containers"
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

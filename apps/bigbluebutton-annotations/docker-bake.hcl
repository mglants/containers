target "docker-metadata-action" {}

variable "APP" {
  default = "bigbluebutton-annotations"
}

variable "VERSION" {
  // renovate: datasource=github-releases depName=bigbluebutton/bigbluebutton versioning=semver
  default = "v4.0.1"
}

variable "SOURCE" {
  default = "https://github.com/bigbluebutton/bigbluebutton"
}

group "default" {
  targets = ["image-local"]
}

target "image" {
  inherits = ["docker-metadata-action"]
  dockerfile = "Dockerfile"
  target = "annotations"
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
  platforms = ["linux/amd64"]
}

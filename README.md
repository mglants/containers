<!---
NOTE: AUTO-GENERATED FILE
to edit this file, instead edit its template at: ./ci/templates/README.md.j2
-->
<div align="center">


## Containers

_A Collection of Container Images Optimized for Kubernetes_

</div>

<div align="center">

![GitHub Repo stars](https://img.shields.io/github/stars/mglants/containers?style=for-the-badge)
![GitHub forks](https://img.shields.io/github/forks/mglants/containers?style=for-the-badge)
![GitHub Workflow Status (with event)](https://img.shields.io/github/actions/workflow/status/mglants/containers/scheduled-release.yaml?style=for-the-badge&label=Scheduled%20Release)

</div>

---

## About

This repo contains a collection of containers which are optimized for use in kubernetes, and updated automatically to keep up with upstream versions. Using an image effectively in Kubernetes requires a few ingredients:

- The filesystem must be able to be immutable
- Semantic versioning is available to specify exact versions to run
- The container can be run rootless
- The container shouldn't require any manual interaction
- The container should ideally be configurable via environmental variables

## Configuration volume

For applications that need to have persistent configuration data the container will leverage a `/data` and a `/config` volume where these are necessary. This is not able to be changed in most cases.

---

## Available Tags

For Semantically Versioned containers (e.g. `v1.2.3`), `major`, `major.minor`, and `major.minor.patch` tags will be generated, for example, ![1](https://img.shields.io/badge/1-blue?style=flat-square) ![1.2](https://img.shields.io/badge/1.2-blue?style=flat-square) and ![1.2.3](https://img.shields.io/badge/1.2.3-blue?style=flat-square). Available Images Below.

### Application Images

Each Image will be built with a `rolling` tag, along with tags specific to it's version. Available Images Below

Container | Channel | Image
--- | --- | ---
[cni-plugins](https://github.com/mglants/containers/pkgs/container/cni-plugins) | stable | ghcr.io/mglants/cni-plugins
[emonoda](https://github.com/mglants/containers/pkgs/container/emonoda) | stable | ghcr.io/mglants/emonoda
[gammu-telegram](https://github.com/mglants/containers/pkgs/container/gammu-telegram) | stable | ghcr.io/mglants/gammu-telegram
[kea-dhcp](https://github.com/mglants/containers/pkgs/container/kea-dhcp) | stable | ghcr.io/mglants/kea-dhcp
[kea-dhcp-sidecar](https://github.com/mglants/containers/pkgs/container/kea-dhcp-sidecar) | stable | ghcr.io/mglants/kea-dhcp-sidecar
[kube-vip-watcher](https://github.com/mglants/containers/pkgs/container/kube-vip-watcher) | stable | ghcr.io/mglants/kube-vip-watcher
[matchbox](https://github.com/mglants/containers/pkgs/container/matchbox) | stable | ghcr.io/mglants/matchbox
[mktxp](https://github.com/mglants/containers/pkgs/container/mktxp) | stable | ghcr.io/mglants/mktxp
[rpi-dnsmasq](https://github.com/mglants/containers/pkgs/container/rpi-dnsmasq) | stable | ghcr.io/mglants/rpi-dnsmasq
[s4cmd](https://github.com/mglants/containers/pkgs/container/s4cmd) | stable | ghcr.io/mglants/s4cmd
[smartctl-exporter](https://github.com/mglants/containers/pkgs/container/smartctl-exporter) | stable | ghcr.io/mglants/smartctl-exporter

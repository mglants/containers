# BigBlueButton 4.0 images

This app builds the BigBlueButton **recorder** component for Linux amd64. Its
Dockerfile includes only this component's dependency stages and source downloads.
`build/` contains its build scripts and patches, when needed; `runtime/` contains
its configuration adapter and required wrappers. The common Go configuration
adapter is kept in every app context so upstream's independent builds work
without shared repository paths or prebuilt custom base images.

## Images

Images are published under `ghcr.io/mglants/` with semantic version and
`rolling` tags. For BBB 4.0.0 these include `4.0.0`, `4.0` and `4`, following
the unchanged upstream publishing workflow.

| Image                            | Purpose                                                                    |
| -------------------------------- | -------------------------------------------------------------------------- |
| bigbluebutton-frontend           | HTML5 client, analytics dashboard, playback and nginx routing.             |
| bigbluebutton-web                | HTTP API, presentation upload and document conversion.                     |
| bigbluebutton-apps               | Meeting state, permissions and application logic.                          |
| bigbluebutton-graphql-actions    | Client mutations and Redis messaging.                                      |
| bigbluebutton-graphql-middleware | Authenticated GraphQL connections.                                         |
| bigbluebutton-notes              | Collaborative notes and document export.                                   |
| bigbluebutton-annotations        | Whiteboard annotation export.                                              |
| bigbluebutton-signaling          | BBB media orchestration for LiveKit.                                       |
| bigbluebutton-bootstrap          | Tools and schemas for explicit database/Hasura initialization.             |
| bigbluebutton-recorder           | LiveKit audio, camera and screen-share capture.                            |
| bigbluebutton-recordings         | Processing, interactive recording publication and downloadable MP4 export. |
| bigbluebutton-webhooks           | Signed event callbacks and optional xAPI integration.                      |
| bigbluebutton-plugins            | Static hosting for the transcription plugin.                               |
| bigbluebutton-sip                | Optional telephone integration using BBB-compatible LiveKit SIP.           |
| bigbluebutton-stt                | Optional Gladia transcription/translation agent.                           |

`recordings` includes MP4 export; separate worker/scheduler/caption processes
share that image. `recordings-base` is an internal build stage, not a published
image.
PostgreSQL, Redis, Hasura, LiveKit and Greenlight use their upstream images.
Provider transcription and SIP require their own credentials/network setup.

## LiveKit compatibility

Recorder v0.17.0 requires LiveKit server 1.13 or newer, as documented in the
[recorder v0.16.0 release notes](https://github.com/bigbluebutton/bbb-webrtc-recorder/releases/tag/v0.16.0).
Use a compatible server when deploying or testing this image.

## Runtime footprint

The runtime uses `scratch` with static Go executables, static Tini, CA
certificates, native configuration paths and a writable `/tmp`. It runs as
UID 10001 and has no shell or package manager. Configuration rendering and
parsing by the recorder are covered by the image tests.

Go executables are built without debug symbols.

## Build and test

From the repository root:

```sh
mise run local-build bigbluebutton-recorder
```

Or build one target directly:

```sh
cd apps/bigbluebutton-recorder
docker buildx bake --load image-local
TEST_IMAGE=bigbluebutton-recorder:v4.0.0 go test -count=1 -v .
```

The per-image Bake definition selects a native Dockerfile target and passes the
normalized `VERSION`. Direct Dockerfile builds must provide `--build-arg
VERSION=4.0.0` and `--target recorder`; there is no default version or CHANNEL selector.
Cold builds download source archives, dependencies and base images.

## Versions and CI

Renovate tracks BBB versions in Bake files and companion source tags in each
Dockerfile, using the upstream Renovate configuration. Merging the PR builds the pinned sources; there is no separate hourly
release discovery or checksum updater. Tagged archives follow the upstream
containers convention; base images retain digest pins. Deploy image digests when
an immutable build is required.

Each app directory builds independently through the upstream workflow. Go container tests
check the selected component, configuration rendering, bundled executables and
non-root execution without requiring backend services. Runtime adapter unit
tests also run during the Docker build. The adapter is a separate Go module so
its production code is isolated from the repository's test module.

The first hosted build still needs validation for runner disk, RAM and duration.
Local image checks do not establish hosted runner capacity.

## Configuration and testing

See [ENVIRONMENT.md](ENVIRONMENT.md) for settings, secret-file inputs and native
configuration mode. Keep API, database and LiveKit credentials stable across
replacement. Database initialization, media storage, TURN, public media ports,
HTTPS routing, backups and resource limits belong to deployment configuration.

Local Compose fixtures, credentials and historical notes live in the ignored
repository directory `.private/bigbluebutton/.local-testing/`. Multi-client media, collaboration, SSO, provider
transcription, carrier audio and public TURN require deployment acceptance tests.
Review companion versions and compatibility patches when upgrading BBB; patches
fail the build when expected upstream code changes.

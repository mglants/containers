FROM golang:1.27-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS build
ARG VERSION=dev
WORKDIR /source
COPY runtime/sync/go.mod runtime/sync/go.sum ./
RUN go mod download
COPY runtime/sync/ ./
RUN go test . && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/remnawave-s3-sync .

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/remnawave-s3-sync /usr/local/bin/remnawave-s3-sync
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/remnawave-s3-sync"]

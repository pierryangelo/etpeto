# syntax=docker/dockerfile:1.7

FROM golang:1.25.7-alpine AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/renderer ./cmd/renderer && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/healthcheck ./cmd/healthcheck

FROM alpine:3.22 AS typst
ARG TARGETARCH
ARG TYPST_VERSION=0.14.2
RUN apk add --no-cache ca-certificates curl tar xz && \
    case "$TARGETARCH" in \
      amd64) target="x86_64-unknown-linux-musl"; sha256="a6044cbad2a954deb921167e257e120ac0a16b20339ec01121194ff9d394996d" ;; \
      arm64) target="aarch64-unknown-linux-musl"; sha256="491b101aa40a3a7ea82a3f8a6232cabb4e6a7e233810082e5ac812d43fdcd47a" ;; \
      *) echo "unsupported architecture: $TARGETARCH" >&2; exit 1 ;; \
    esac && \
    archive="typst-${target}.tar.xz" && \
    curl --fail --location --output "/tmp/${archive}" \
      "https://github.com/typst/typst/releases/download/v${TYPST_VERSION}/${archive}" && \
    echo "${sha256}  /tmp/${archive}" | sha256sum -c - && \
    tar -xJf "/tmp/${archive}" -C /tmp && \
    cp "/tmp/typst-${target}/typst" /out-typst && \
    chmod 0555 /out-typst

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=go-build /out/renderer /app/renderer
COPY --from=go-build /out/healthcheck /app/healthcheck
COPY --from=typst /out-typst /usr/local/bin/typst
USER nonroot:nonroot
ENTRYPOINT ["/app/renderer"]

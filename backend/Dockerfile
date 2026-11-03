# syntax=docker/dockerfile:1

# ---- build ---------------------------------------------------------------
FROM golang:1.27-alpine AS build
WORKDIR /src

# Cache module downloads separately from source changes.
COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath \
      -ldflags="-s -w -X $(go list -m)/internal/httpapi.Version=${VERSION}" \
      -o /out/server ./cmd/server

# ---- runtime -------------------------------------------------------------
# Static, non-root, CA certificates and tzdata included, no shell.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
ENV BACKEND_ADDR=:8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]

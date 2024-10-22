# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS builder
ARG VERSION
RUN apk add --no-cache gcc sqlite-dev musl-dev
WORKDIR /build
# First only add go.mod and go.sum, then run go mod download to cache dependencies
# in a separate layer.
ADD go.mod go.sum /build/
RUN --mount=type=cache,target=/root/.cache/go-build go mod download
# Now add the rest of the source code and build the application.
ADD . /build/
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=1 GOOS=linux \
    go build \
        -trimpath \
        -ldflags "-s -X github.com/flashbots/builder-hub/common.Version=${VERSION} -w -extldflags \"-static\"" \
        -v \
        -o builder-hub \
    cmd/httpserver/main.go

FROM alpine:latest
RUN apk update && apk upgrade
RUN apk add --no-cache sqlite-dev
# See http://stackoverflow.com/questions/34729748/installed-go-binary-not-found-in-path-on-alpine-linux-docker
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/builder-hub /app/builder-hub
ADD testdata/ /app/testdata/
CMD ["/app/builder-hub"]

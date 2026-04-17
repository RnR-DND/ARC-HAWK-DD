# syntax=docker/dockerfile:1.6

FROM golang:1.22-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/scanner ./cmd/scanner

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget tzdata && adduser -D -u 10001 scanner
WORKDIR /app
COPY --from=builder /out/scanner /app/scanner
USER scanner
EXPOSE 8001
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD wget -qO- http://localhost:8001/health || exit 1
ENTRYPOINT ["/app/scanner"]

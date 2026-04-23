FROM golang:1.22-bookworm AS builder

RUN apt-get update && apt-get install -y gcc sqlite3 libsqlite3-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /hawk-agent ./cmd/agent

# --- Runtime ---
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates libsqlite3-0 && rm -rf /var/lib/apt/lists/*
RUN useradd -r -s /usr/sbin/nologin hawk

COPY --from=builder /hawk-agent /usr/local/bin/hawk-agent

RUN mkdir -p /var/lib/hawk-agent && chown hawk:hawk /var/lib/hawk-agent
VOLUME ["/var/lib/hawk-agent"]

ENV HAWK_DATA_DIR=/var/lib/hawk-agent
ENV HAWK_LOG_LEVEL=info

EXPOSE 9090

# The agent needs root for EDR scanning; run as root in container.
USER root

# Required capabilities for EDR scanning documented below.
# All others dropped as defence-in-depth.
# SECURITY: This container requires root for host-level EDR scanning.
# When running in Kubernetes, apply:
#   securityContext:
#     capabilities:
#       drop: ["ALL"]
#       add: ["SYS_PTRACE", "NET_ADMIN"]  # Only if required for scanning
#     readOnlyRootFilesystem: true

ENTRYPOINT ["/usr/local/bin/hawk-agent"]

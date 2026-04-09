# ---------- build stage ----------
FROM golang:1.22-bookworm AS builder

RUN apt-get update && apt-get install -y gcc sqlite3 libsqlite3-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-X main.Version=${VERSION}" \
    -o /hawk-agent \
    ./cmd/agent

# ---------- runtime stage ----------
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates libsqlite3-0 && rm -rf /var/lib/apt/lists/*

RUN useradd --system --no-create-home hawk

COPY --from=builder /hawk-agent /usr/local/bin/hawk-agent

# Default data directory
RUN mkdir -p /var/lib/hawk-agent && chown hawk:hawk /var/lib/hawk-agent
ENV HAWK_DATA_DIR=/var/lib/hawk-agent

EXPOSE 9090

USER hawk
ENTRYPOINT ["/usr/local/bin/hawk-agent"]

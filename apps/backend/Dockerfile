FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

# Final stage
FROM alpine:3.20

# Run as non-root user — principle of least privilege
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

RUN apk --no-cache add ca-certificates

# Copy the binary and migrations from the builder stage
COPY --from=builder /app/main .
COPY --from=builder /app/migrations_versioned ./migrations_versioned

# Set ownership before switching user
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["./main"]

FROM golang:1.24-alpine AS builder

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
FROM alpine:latest

WORKDIR /root/

# Install migration tools if needed, or just standard libs
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Copy migrations (assuming they are needed at runtime for auto-migration)
COPY --from=builder /app/migrations_versioned ./migrations_versioned

# Expose the API port
EXPOSE 8080

# Run the binary
CMD ["./main"]

# Multi-stage build for optimized image size

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN go build \
    -o ipwatcher \
    ./cmd/ipwatcher

# Final stage
FROM scratch

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy SSL certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /build/ipwatcher /ipwatcher

# Copy example config (user should mount their own)
COPY --from=builder /build/config.yaml.example /config.yaml.example

# Set working directory
WORKDIR /

# Set environment variables
ENV CONFIG_FILE=/config/config.yaml

# Run as non-root user (note: scratch doesn't have users, so we rely on --user flag at runtime)
# The container should be run with: docker run --user 1000:1000

# Expose no ports (this is a client application, not a server)

# Run the daemon
ENTRYPOINT ["/ipwatcher"]

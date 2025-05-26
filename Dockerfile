# HyprKnot Dockerfile
# Lightweight HTTP API wrapper for KnotDNS

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.appVersion=1.0.0" \
    -a -installsuffix cgo \
    -o hyprknot .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates knot-utils

# Create non-root user
RUN addgroup -g 1000 hyprknot && \
    adduser -D -s /bin/sh -u 1000 -G hyprknot hyprknot

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/hyprknot /usr/local/bin/hyprknot

# Copy configuration
COPY config.yaml /etc/hyprknot/config.yaml

# Create directories
RUN mkdir -p /var/log/hyprknot && \
    chown -R hyprknot:hyprknot /var/log/hyprknot /etc/hyprknot

# Switch to non-root user
USER hyprknot

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["hyprknot", "-config", "/etc/hyprknot/config.yaml"]

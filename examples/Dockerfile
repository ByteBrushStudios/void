# Multi-stage build for Void Maintenance Server
FROM golang:1.21-alpine AS builder

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

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always)" \
    -o void main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl tzdata && \
    addgroup -g 1000 void && \
    adduser -u 1000 -G void -s /bin/sh -D void

# Set working directory
WORKDIR /app

# Copy binary and configuration
COPY --from=builder /build/void .
COPY --from=builder /build/services ./services/
COPY --from=builder /build/app.html .
COPY --from=builder /build/assets ./assets/

# Set ownership
RUN chown -R void:void /app

# Switch to non-root user
USER void

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:${VOID_PORT:-1292}/health || exit 1

# Expose port
EXPOSE 1292

# Run the application
CMD ["./void"]

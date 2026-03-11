# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /icooclaw ./cmd/icooclaw

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata sqlite

# Create non-root user
RUN adduser -D -g '' appuser

# Copy binary from builder
COPY --from=builder /icooclaw /app/icooclaw

# Create data directory
RUN mkdir -p /app/data /app/workspace && chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Set environment variables
ENV ICOOCLAW_WORKSPACE=/app/workspace

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Entry point
ENTRYPOINT ["/app/icooclaw"]
CMD ["start"]
CMD ["start"]
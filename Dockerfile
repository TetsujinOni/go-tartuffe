# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for fetching dependencies (some may require it)
RUN apk add --no-cache git

# Copy go.mod and go.sum first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o tartuffe ./cmd/tartuffe

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS support
RUN apk add --no-cache ca-certificates

# Create non-root user for security
RUN adduser -D -u 1000 tartuffe

# Copy the binary from builder
COPY --from=builder /app/tartuffe /app/tartuffe

# Create directories for data persistence and logs
RUN mkdir -p /app/data /app/logs && chown -R tartuffe:tartuffe /app

USER tartuffe

# Default port
EXPOSE 2525

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:2525/ || exit 1

ENTRYPOINT ["/app/tartuffe"]

# Default arguments (can be overridden)
CMD ["--host", "0.0.0.0"]

# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy Go module files first for better layer caching
COPY go.mod ./

# Copy source and download/verify dependencies
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o planit .

# Production stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS and wget for healthcheck
RUN apk add --no-cache ca-certificates wget tzdata

# Copy compiled binary from builder
COPY --from=builder /app/planit .

# Copy templates and static assets
COPY templates ./templates
COPY static ./static

# Create data directory for SQLite database
RUN mkdir -p /app/data

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null http://localhost:3000/ || exit 1

# Run the Go application
CMD ["./planit"]

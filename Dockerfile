# Build stage
FROM golang:1.22-alpine AS builder

# Install security updates and ca-certificates
RUN apk update && apk upgrade && apk add --no-cache ca-certificates git

# Create non-root user for runtime
RUN addgroup -g 1001 appgroup && adduser -u 1001 -G appgroup -D appuser

WORKDIR /build

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -trimpath \
    -o /build/server \
    ./cmd/server

# Runtime stage - distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot

# Copy ca-certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /build/server /server

# Copy migrations for runtime (if using embedded migrations)
COPY --from=builder /build/migrations /migrations

# Use non-root user
USER 1001:1001

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/server", "-health-check"] || exit 1

ENTRYPOINT ["/server"]

# ==============================================================================
# Multi-Stage Dockerfile for Gosling MCP Server
# ==============================================================================

# Stage 1: Build binary
FROM golang:1.26-alpine AS builder

ARG VERSION=1.0.0

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X 'github.com/sourabhmandal/gosling/internal/infrastructure/config.Version=${VERSION}'" \
    -o /app/bin/gosling-mcp-server \
    ./cmd/server

# Stage 2: Minimal runtime image
FROM alpine:3.20

ARG VERSION=1.0.0
ENV APP_VERSION=${VERSION}

WORKDIR /app

# Install runtime dependencies (CA certificates for outbound HTTPS calls to Google APIs)
RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy compiled binary from builder stage
COPY --from=builder /app/bin/gosling-mcp-server /app/gosling-mcp-server

# Run as unprivileged user
USER appuser

# Expose MCP server port
EXPOSE 8080

# Run MCP server
ENTRYPOINT ["/app/gosling-mcp-server"]

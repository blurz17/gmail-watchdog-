# Build stage
FROM golang:alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go.mod and go.sum first for better layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy source code.
COPY . .

# Build binaries into /app (same WORKDIR, cleaner paths).
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ./gmail-monitor ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ./gmail-monitor-cli ./cmd/cli

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create a non-root user.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binaries and migrations.
COPY --from=builder /app/gmail-monitor .
COPY --from=builder /app/gmail-monitor-cli .
COPY --from=builder /app/migrations ./migrations

# Grant ownership to non-root user before switching.
RUN chown -R appuser:appgroup /app

# Switch to non-root user.
USER appuser

EXPOSE 8080

ENTRYPOINT ["./gmail-monitor"]

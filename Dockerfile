# -- Build stage --
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bot ./cmd/bot/

# -- Runtime stage --
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata ffmpeg

# Non-root user
RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /bot .

# Data directory for SQLite
RUN mkdir -p /app/data && chown -R app:app /app

USER app

EXPOSE 8080

ENV PORT=8080 \
    ENV=production

ENTRYPOINT ["./bot"]

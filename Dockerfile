# ── Stage 1: Build Vue frontend ──
FROM node:22-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ .
RUN npm run build

# ── Stage 2: Build Go backend (embed Vue dist) ──
FROM golang:1.25-alpine AS backend-builder

WORKDIR /src

# Cache Go deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Inject the built Vue dist into the SPA embed directory
COPY --from=frontend-builder /app/frontend/dist ./internal/spa/dist

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server/

# ── Stage 3: Minimal runtime ──
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=backend-builder /server .

# Uploads directory (Railway volume or ephemeral)
RUN mkdir -p /app/uploads && chown -R app:app /app

USER app

ENV ENV=production
EXPOSE 3000

ENTRYPOINT ["./server"]

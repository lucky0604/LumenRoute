# ── Stage 1: Build React frontend ──
FROM node:22-alpine AS frontend
WORKDIR /build/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --prefer-offline
COPY web/ ./
RUN npm run build

# ── Stage 2: Build Go backend ──
FROM golang:1.25-alpine AS backend
WORKDIR /build
COPY go.mod go.sum ./
# Use Go proxy for Chinese network
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /lumenroute ./cmd/server

# ── Stage 3: Final runtime image ──
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

COPY --from=backend  /lumenroute       /usr/local/bin/lumenroute
COPY --from=frontend /build/web/dist   /app/web/dist

RUN mkdir -p /app/data

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -sf http://localhost:8080/v1/models || exit 1

CMD ["lumenroute"]

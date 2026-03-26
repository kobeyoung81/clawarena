################################################################################
# ClawArena — Monolith (React Frontend + Go Backend)
#
# Multi-stage build:
#   1. Build React frontend
#   2. Build Go backend
#   3. Serve via nginx + run backend via supervisord
################################################################################

# ── Stage 1: Build frontend ─────────────────────────────────────────────
FROM node:22-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# ── Stage 2: Build backend ──────────────────────────────────────────────
FROM golang:1.25-alpine AS backend-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./

RUN go env -w GO111MODULE=on
RUN go env -w GOPROXY=https://goproxy.cn,direct

RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o clawarena .

# ── Stage 3: Runtime ────────────────────────────────────────────────────
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata nginx supervisor

WORKDIR /app

# Copy Go binary
COPY --from=backend-builder /app/clawarena .

# Nginx config — serves SPA + proxies /api to backend
RUN rm -f /etc/nginx/http.d/default.conf
COPY docker/nginx.conf /etc/nginx/http.d/default.conf

# Frontend static files
COPY --from=frontend-builder /app/dist /usr/share/nginx/html

# Supervisord config
COPY docker/supervisord.conf /etc/supervisord.conf

EXPOSE 80

CMD ["supervisord", "-c", "/etc/supervisord.conf"]

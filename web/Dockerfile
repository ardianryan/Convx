# ==============================================================================
# Convx Music — Multi-Stage Docker Build (Optimized Caching)
# Architecture: Node.js Gateway (port 7554) → Go Backend (port 7555)
# ==============================================================================

# Stage 1: Build Frontend (Svelte 5 + Vite + Tailwind CSS)
FROM node:22-alpine AS frontend-builder
WORKDIR /build/frontend
COPY frontend/package*.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --ignore-scripts
COPY frontend/ ./
RUN npm run build
# Output: /build/backend/dist (vite.config.js outDir: '../backend/dist')

# Stage 2: Build Go Backend (Static Binary)
FROM golang:1.24-alpine AS go-builder
ENV GOTOOLCHAIN=auto
WORKDIR /build/backend
COPY backend/go.mod backend/go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY backend/ ./
COPY --from=frontend-builder /build/backend/dist ./dist
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /convx-go .

# Stage 3: Install Node.js Gateway Dependencies
FROM node:22-alpine AS node-builder
RUN apk add --no-cache python3 make g++
WORKDIR /build/server
COPY server/package*.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --omit=dev
COPY server/ ./

# Stage 4: Production Image
FROM node:22-alpine

RUN apk --no-cache add ca-certificates tzdata wget libstdc++

# Create non-root user
RUN addgroup -S convx && adduser -S convx -G convx

WORKDIR /app

# Copy Go binary
COPY --from=go-builder /convx-go /app/convx-go

# Copy Node.js gateway (with node_modules)
COPY --from=node-builder /build/server /app/server

# Copy frontend dist for Node.js static serving
COPY --from=frontend-builder /build/backend/dist /app/backend/dist

# Copy entrypoint and env example
COPY entrypoint.sh /app/entrypoint.sh
COPY .env.example /app/.env.example

# Create data directory for SQLite and set permissions
RUN chmod +x /app/entrypoint.sh \
    && mkdir -p /app/backend/data \
    && chown -R convx:convx /app

USER convx

ENV CONVX_PORT=7554
ENV GO_BACKEND_PORT=7555

EXPOSE 7554

VOLUME ["/app/backend/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null "http://localhost:${CONVX_PORT:-7554}/api/health" || exit 1

ENTRYPOINT ["/app/entrypoint.sh"]

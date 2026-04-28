# Multi-stage Dockerfile for Brewmaster API + PWA
# Build from repo root: docker build -t brewmaster .

# Stage 1: Build React PWA
FROM node:22-alpine AS pwa-builder

WORKDIR /pwa

COPY pwa/package*.json ./
RUN npm ci

COPY pwa/ ./

# Build PWA (vite.config.ts outputs to ../api/static for local dev).
# Override outDir here since the Docker context has a different directory layout.
RUN mkdir -p /static && \
    sed -i "s|outDir: '../api/static'|outDir: '/static'|" vite.config.ts && \
    npm run build

# Stage 2: Build Go API
FROM golang:1.26-alpine AS go-builder

WORKDIR /app

COPY api/go.mod api/go.sum* ./
RUN go mod download

COPY api/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Stage 3: Minimal runtime image
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=go-builder /app/server .
COPY --from=pwa-builder /static ./static

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["./server"]

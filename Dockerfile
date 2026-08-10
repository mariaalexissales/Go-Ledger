# syntax=docker/dockerfile:1

# --- Stage 1: build the React console -----------------------------------------
FROM node:22-alpine AS web
WORKDIR /web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# --- Stage 2: build the Go binary with the console embedded -------------------
FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# vite.config.ts writes to ../internal/spa/dist, which resolves to
# /internal/spa/dist in the web stage. Copy it where //go:embed expects it.
COPY --from=web /internal/spa/dist ./internal/spa/dist

RUN CGO_ENABLED=0 go build -tags embed_spa -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# --- Stage 3: runtime ---------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/server /server

EXPOSE 8080
USER nonroot:nonroot

# Exec form, because distroless has no shell. `server healthcheck` is a
# subcommand that hits /health and exits 0 or 1.
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/server", "healthcheck"]

ENTRYPOINT ["/server"]

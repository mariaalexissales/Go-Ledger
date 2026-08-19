# syntax=docker/dockerfile:1

# Kept in step with .nvmrc and the CI workflows, which read the same file.
ARG NODE_VERSION=22

# --- Stage 1: build the React console -----------------------------------------
FROM node:${NODE_VERSION}-alpine AS web
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

# Just the Go tree. Copying everything would drag web/ in for nothing, since the
# built console arrives from the web stage below rather than being built here.
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# vite.config.ts writes to ../internal/spa/dist, which resolves to
# /internal/spa/dist in the web stage. Copy it where //go:embed expects it.
COPY --from=web /internal/spa/dist ./internal/spa/dist

RUN CGO_ENABLED=0 go build -tags embed_spa -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# --- Stage 3: runtime ---------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="go-ledger" \
      org.opencontainers.image.description="Ledger API demonstrating IP rate limiting and X-Forwarded-For trust" \
      org.opencontainers.image.source="https://github.com/mariaalexissales/Go-Ledger" \
      org.opencontainers.image.licenses="MIT"

COPY --from=build /out/server /server

EXPOSE 8080
USER nonroot:nonroot

# Exec form, because distroless has no shell. `server healthcheck` is a
# subcommand that hits /health and exits 0 or 1.
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/server", "healthcheck"]

ENTRYPOINT ["/server"]

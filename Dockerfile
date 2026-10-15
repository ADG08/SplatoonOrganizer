# Build: a static binary (the timezones' data is embedded by the code).
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# Built for the host arch (the Coolify VPS may be arm64, a local machine amd64).
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bot ./cmd/bot

# Run: distroless holds only the CA certificates (for Discord's HTTPS) and a non-root
# user: no shell nor package manager to attack, a few MB.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bot /bot

# No wget in the image: the bot checks its own /health.
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 CMD ["/bot", "healthcheck"]

ENTRYPOINT ["/bot"]

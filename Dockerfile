FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build for host arch (Coolify VPS is often aarch64; local may be amd64/arm64).
RUN CGO_ENABLED=0 GOOS=linux go build -o bot ./cmd/bot

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/bot /app/bot

ENV DISCORD_TOKEN=""
ENV DISCORD_CLIENT_ID=""
ENV DISCORD_GUILD_ID=""
ENV DISCORD_CHANNEL_ID=""
ENV DATABASE_URL=""
ENV TZ="Europe/Paris"

CMD ["/app/bot"]


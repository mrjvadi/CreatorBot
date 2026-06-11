FROM golang:1.22-alpine AS builder
WORKDIR /workspace
# Copy workspace root for go.work shared module resolution
COPY go.work ./
COPY go.work.sum ./
COPY shared/ shared/
COPY vpn-bot/ vpn-bot/
WORKDIR /workspace/vpn-bot
RUN go build -o /out/bot ./cmd/bot

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/bot .
ENTRYPOINT ["/app/bot"]

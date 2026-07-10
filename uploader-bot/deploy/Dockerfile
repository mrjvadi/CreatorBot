FROM golang:1.25-alpine AS builder
WORKDIR /workspace

# Copy entire workspace so go.work can find all required modules
COPY . .

WORKDIR /workspace/uploader-bot
RUN go build -o /out/bot ./cmd/bot

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/bot .
ENTRYPOINT ["/app/bot"]

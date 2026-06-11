FROM golang:1.22-alpine AS builder
WORKDIR /workspace
COPY go.work ./
COPY go.work.sum ./
COPY shared/ shared/
COPY source-service/ source-service/
WORKDIR /workspace/source-service
RUN go build -o /out/service ./cmd/service

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/service .
VOLUME ["/app/sessions"]
ENTRYPOINT ["/app/service"]

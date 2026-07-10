FROM golang:1.25-alpine AS builder
WORKDIR /workspace
COPY go.work ./
COPY go.work.sum ./
COPY shared/ shared/
COPY shared-core/ shared-core/
COPY source-service/ source-service/
WORKDIR /workspace/source-service
RUN go build -o /out/service ./cmd/service
RUN go build -o /out/login ./cmd/login

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/service .
COPY --from=builder /out/login .
VOLUME ["/app/sessions"]
ENTRYPOINT ["/app/service"]

FROM golang:1.25-alpine AS builder
WORKDIR /workspace
COPY go.work ./
COPY go.work.sum ./
COPY shared/ shared/
COPY shared-core/ shared-core/
COPY apimanager/ apimanager/
WORKDIR /workspace/apimanager
RUN go build -o /out/app ./cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/app .
ENTRYPOINT ["/app/app"]

FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum ./
COPY shared/ shared/
COPY shared-core/ shared-core/
COPY agentmanager/ agentmanager/
RUN go build -o /bin/service ./agentmanager/cmd/...

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/service /service
ENTRYPOINT ["/service"]

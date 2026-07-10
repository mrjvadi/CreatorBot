FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum ./
COPY shared/ shared/
COPY image-registry/ image-registry/
RUN go build -o /bin/service ./image-registry/cmd/...

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/service /service
# باید در production یک volume پایدار روی همین مسیر mount شود
# (IMAGE_STORAGE_DIR) وگرنه فایل‌های image با هر redeploy از دست می‌روند.
VOLUME ["/data/images"]
ENTRYPOINT ["/service"]

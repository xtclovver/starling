FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY go.work go.work
COPY gen/go/go.mod gen/go/go.sum* gen/go/
COPY api-gateway/go.mod api-gateway/go.sum* api-gateway/
COPY services/user-svc/go.mod services/user-svc/go.sum* services/user-svc/
COPY services/post-svc/go.mod services/post-svc/go.sum* services/post-svc/
COPY services/comment-svc/go.mod services/comment-svc/go.sum* services/comment-svc/
COPY services/media-svc/go.mod services/media-svc/go.sum* services/media-svc/

RUN cd services/user-svc && go mod download

COPY gen/ gen/
COPY services/user-svc/ services/user-svc/

RUN cd services/user-svc && CGO_ENABLED=0 GOOS=linux go build -o /app ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /app /app

EXPOSE 50051

ENTRYPOINT ["/app"]

FROM golang:1.23-alpine AS builder

WORKDIR /build

COPY go.work go.work
COPY api-gateway/go.mod api-gateway/go.sum* api-gateway/
COPY services/user-svc/go.mod services/user-svc/go.sum* services/user-svc/
COPY services/post-svc/go.mod services/post-svc/go.sum* services/post-svc/
COPY services/comment-svc/go.mod services/comment-svc/go.sum* services/comment-svc/
COPY services/media-svc/go.mod services/media-svc/go.sum* services/media-svc/

RUN cd services/comment-svc && go mod download

COPY gen/ gen/
COPY services/comment-svc/ services/comment-svc/

RUN cd services/comment-svc && CGO_ENABLED=0 GOOS=linux go build -o /app ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /app /app

EXPOSE 50053

ENTRYPOINT ["/app"]

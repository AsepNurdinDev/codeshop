# Build Stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/auth-service ./cmd/server

# Final Stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/auth-service .
COPY --from=builder /app/migrations ./migrations

USER appuser

EXPOSE 50051

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD nc -z localhost 50051 || exit 1

ENTRYPOINT ["/app/auth-service"]

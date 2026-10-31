# syntax=docker/dockerfile:1

# --- Build stage ---
# Compiles a static binary so the final image can be a minimal distroless
# runtime with no Go toolchain, shell, or package manager
# (small attack surface, per ADR-010's Kubernetes/security-primitives intent).
FROM golang:1.22-bookworm AS builder

WORKDIR /src

# Cache module downloads separately from source so `docker build` only
# re-downloads dependencies when go.mod/go.sum actually change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 -> fully static binary, safe to run on a scratch/distroless
# base with no libc.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/auth-service ./cmd/server

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app
COPY --from=builder /out/auth-service /app/auth-service
COPY --from=builder /src/migrations /app/migrations

# distroless/static:nonroot already runs as a non-root "nonroot" user.
USER nonroot:nonroot

EXPOSE 8080 9090

ENTRYPOINT ["/app/auth-service"]

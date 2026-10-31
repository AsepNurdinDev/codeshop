# CodeShop Auth Service

Authentication and authorization microservice for the CodeShop digital marketplace.

The Auth Service is responsible for user identity, authentication, authorization, access tokens, refresh tokens, and user account management.

This service is designed as an independently deployable microservice and is intended to run locally, inside Docker, and eventually on Kubernetes in a production environment.

---

## 1. Responsibility

The Auth Service owns the following responsibilities:

- User registration
- User authentication
- User login
- Password hashing and verification
- JWT access token generation
- Refresh token management
- Token rotation
- Logout / token revocation
- User identity retrieval
- Role-based authorization
- User account status
- Authentication-related security controls
- Authentication audit events

The Auth Service does **not** own:

- Product catalog
- Shopping cart
- Orders
- Payments
- Digital downloads
- Notifications
- Shipping

Those responsibilities belong to other CodeShop services.

---

## 2. Service Boundary

The Auth Service owns user identity and authentication data.

```text
                    CodeShop Platform
                           │
                           ▼
                    ┌──────────────┐
                    │ API Gateway  │
                    └──────┬───────┘
                           │
                           ▼
                  ┌──────────────────┐
                  │   Auth Service   │
                  └────────┬─────────┘
                           │
                ┌──────────┴──────────┬─────────────────┐
                ▼                     ▼                  ▼
          PostgreSQL                Redis            RabbitMQ
        (users, refresh      (active refresh      (UserRegistered
         token audit)        token sessions,        event → Notification
                              login rate limit)       Service)
```

Other services (API Gateway, Order Service, Download Service, ...) never
query the Auth DB directly. They talk to Auth Service over:

- **gRPC (internal)** — `ValidateToken`, `GetUser` (see `proto/auth.proto`)
- **REST (public)** — `/api/v1/auth/*`, `/api/v1/users/me`, served directly
  today and intended to sit behind an API Gateway once that service exists

---

## 3. Tech Stack

| Concern              | Choice                                             |
|-----------------------|----------------------------------------------------|
| Language              | Go 1.22                                             |
| HTTP router           | [chi](https://github.com/go-chi/chi)                |
| Internal RPC          | gRPC (`google.golang.org/grpc`) + Protocol Buffers  |
| Database              | PostgreSQL (via `pgx/v5`)                           |
| Cache / sessions      | Redis (via `go-redis/v9`)                           |
| Messaging             | RabbitMQ (via `amqp091-go`)                         |
| Password hashing      | Argon2id (`golang.org/x/crypto/argon2`)             |
| Access tokens         | JWT, HS256 (`golang-jwt/jwt/v5`)                    |

---

## 4. Project Layout

```text
auth-service/
├── cmd/server/main.go        # entrypoint: wires config, DB, Redis, RabbitMQ, HTTP + gRPC servers
├── internal/
│   ├── config/                # env-var configuration loading
│   ├── domain/                 # core entities: User, Role, Token, sentinel errors
│   ├── repository/             # PostgreSQL (users, refresh-token audit) + Redis (sessions)
│   ├── service/                 # business logic: auth_service.go, token_service.go
│   ├── event/                    # RabbitMQ UserRegistered publisher
│   ├── handler/                   # REST handlers (chi)
│   ├── middleware/                 # request ID, panic recovery, JWT auth
│   ├── grpcserver/                  # gRPC server implementing proto/auth.proto
│   └── webutil/                      # shared JSON response + request-context helpers
├── pkg/pb/                    # generated gRPC/protobuf code (from proto/auth.proto)
├── proto/auth.proto           # internal gRPC contract
├── migrations/                # SQL migrations (users, refresh_tokens)
├── tests/                      # unit tests (auth_test.go) + HTTP integration tests (integration_test.go)
├── Dockerfile                  # production image (multi-stage, distroless)
├── Dockerfile.dev               # development image (hot reload via air)
├── docker-compose.yml            # auth-service + Postgres + Redis + RabbitMQ, for local dev
└── Makefile
```

---

## 5. Running the Service

There are three ways to run Auth Service locally, from least to most setup.

### Option A — Docker Compose (recommended, least setup)

This starts Auth Service **together with its own PostgreSQL, Redis, and
RabbitMQ** — no local Go/Postgres/Redis install required. This is the
fastest way to get a fully working instance.

```bash
cd auth-service
docker compose up --build
```

This will:
- build the dev image (`Dockerfile.dev`, with hot-reload via `air`)
- start PostgreSQL (`localhost:5432`), Redis (`localhost:6379`), and
  RabbitMQ (`localhost:5672`, management UI at `localhost:15672`,
  user/pass `guest`/`guest`)
- start Auth Service on `localhost:8080` (REST) and `localhost:9090` (gRPC)

Apply the database migrations once the Postgres container is healthy
(see §6 below), then verify the service is up:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

Stop everything with:

```bash
docker compose down -v   # -v also removes the Postgres data volume
```

### Option B — `make run` / `go run` against local infra

If you already have PostgreSQL, Redis, and RabbitMQ running locally (or
want to point at remote instances):

```bash
cd auth-service
cp .env.example .env   # then edit .env with your connection strings
go mod download
make migrate-up         # requires the golang-migrate CLI, see §6
make run                 # equivalent to: go run ./cmd/server
```

`.env` is loaded automatically on startup (see `cmd/server/main.go`) and
is git-ignored — never commit real secrets in it.

### Option C — build a binary

```bash
cd auth-service
make build               # outputs ./bin/auth-service
DATABASE_URL=... REDIS_ADDR=... RABBITMQ_URL=... JWT_SECRET=... ./bin/auth-service
```

> **Production note:** `JWT_SECRET` must be set explicitly when
> `APP_ENV=production` — the service refuses to start with an empty
> secret in that mode (see `internal/config/config.go`). In development,
> an insecure default is used if unset, purely for `go run` convenience.

---

## 6. Database Migrations

Migrations live in `migrations/` and follow the
[golang-migrate](https://github.com/golang-migrate/migrate) `NNN_name.up.sql`
/ `NNN_name.down.sql` naming convention.

Install the CLI (one-time):

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Apply migrations (reads `DATABASE_URL` from your shell env, matching
`.env`):

```bash
export DATABASE_URL=postgres://auth:auth@localhost:5432/auth_db?sslmode=disable
make migrate-up
```

Roll back the last migration:

```bash
make migrate-down
```

If you're running via Docker Compose, run the same commands from the host
with `DATABASE_URL=postgres://auth:auth@localhost:5432/auth_db?sslmode=disable`
(the `5432` port is published to the host).

---

## 7. Testing

```bash
make test            # go test ./...
make test-verbose    # -v output
make test-cover       # with coverage report
```

- **`tests/auth_test.go`** — unit tests for `AuthService` business logic
  (register/login/refresh/logout, password hashing, JWT issuance/validation,
  rate-limit fail-open behavior), using in-memory fakes for the repository
  interfaces. No external services required.
- **`tests/integration_test.go`** — exercises the full REST stack
  (router → middleware → handler → service) with `httptest`, against the
  same in-memory fakes. Also no external services required.

Both suites run in a few seconds with zero setup (`go test ./...`), which
is deliberate: they validate application logic and HTTP wiring, not
infrastructure connectivity. A true infrastructure-integration suite
(against real Postgres/Redis via Docker Compose) is a natural next step
once this service is deployed alongside the rest of the platform.

All 17 tests pass as of this implementation (`go test ./... -v`).

---

## 8. API Overview

REST (public, `/api/v1`):

| Method | Path              | Auth required | Description |
|--------|-------------------|:---:|-------------|
| POST   | `/auth/register`  |  – | Create a new buyer account |
| POST   | `/auth/login`     |  – | Authenticate, receive access + refresh tokens |
| POST   | `/auth/refresh`   |  – | Exchange a valid refresh token for a new pair (rotates it) |
| POST   | `/auth/logout`    |  – | Revoke a refresh token (idempotent) |
| GET    | `/users/me`       | ✅ | Get the authenticated caller's own profile |
| GET    | `/healthz`        |  – | Liveness probe |
| GET    | `/readyz`         |  – | Readiness probe (checks Postgres + Redis) |

Authenticated requests use `Authorization: Bearer <access_token>`.

gRPC (internal, port `9090`, see `proto/auth.proto`):

| RPC            | Description |
|-----------------|-------------|
| `ValidateToken` | Stateless JWT validation, for other services to trust a caller's identity |
| `GetUser`       | Look up public identity fields by `user_id` |

Full request/response contracts: see `docs/architecture/API_DESIGN.md` and
`proto/auth.proto`.

---

## 9. Configuration Reference

All configuration is via environment variables (see `.env.example` and
`internal/config/config.go` for defaults):

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | `production` enforces `JWT_SECRET` |
| `HTTP_PORT` | `8080` | REST listen port |
| `GRPC_PORT` | `9090` | gRPC listen port |
| `DATABASE_URL` | `postgres://auth:auth@localhost:5432/auth_db?sslmode=disable` | PostgreSQL DSN |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | RabbitMQ connection string |
| `RABBITMQ_EXCHANGE` | `codeshop.events` | Shared topic exchange for domain events |
| `JWT_SECRET` | *(dev fallback only)* | HMAC signing secret for access tokens |
| `ACCESS_TOKEN_TTL` | `15m` | Access token lifetime |
| `REFRESH_TOKEN_TTL` | `168h` (7 days) | Refresh token lifetime |
| `LOGIN_RATE_LIMIT_ATTEMPTS` | `5` | Failed logins allowed per window before lockout |
| `LOGIN_RATE_LIMIT_WINDOW` | `15m` | Rate-limit window |

If RabbitMQ is unreachable at startup, Auth Service logs a warning and
continues without publishing `UserRegistered` events — registration and
login are never blocked by the message broker being down.

---

## 10. Regenerating gRPC Code

If `proto/auth.proto` changes, regenerate `pkg/pb/`:

```bash
make proto
```

Requires `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc` on your
`PATH`.

---

## 11. Design Notes

- **Refresh tokens live in two places on purpose**: Redis is the source of
  truth for whether a token is *currently* valid (fast lookup, TTL-based
  expiry, O(1) revocation on logout). PostgreSQL keeps a durable audit
  trail of every token issued/rotated/revoked, used for security
  investigations and refresh-token-reuse detection. See the comment block
  in `migrations/002_create_refresh_tokens.up.sql`.
- **Access tokens are stateless JWTs** so other services (and, later, the
  API Gateway) can validate a token's signature and expiry without a
  round-trip to Auth Service — this keeps token validation working even
  during a brief Auth Service outage.
- **Login rate limiting fails open**: if Redis is unreachable, login is
  allowed rather than blocked, so a Redis outage degrades security posture
  slightly rather than taking down authentication entirely.
- **`UserRegistered` publish is best-effort**: a RabbitMQ failure never
  fails the registration request itself; Notification Service simply
  won't send a welcome email for that user until the issue is resolved.

See `docs/architecture/` and `docs/decisions/` at the monorepo root for
the full architectural context (ADR-003 gRPC, ADR-005 RabbitMQ, ADR-006
PostgreSQL, ADR-007 Redis).

---

## 12. Known Limitations / Not Yet Implemented

- No password-reset flow (forgot password) yet.
- No admin endpoints for suspending/reinstating a user account yet
  (the `status` column and domain logic support it; no HTTP route exists).
- No "logout of all devices" endpoint yet (`RevokeAllForUser` exists at
  the repository layer, not wired to a handler).
- No structured/JSON logging — currently uses the standard `log` package.
  Swap-in point is `cmd/server/main.go` and `internal/middleware/recovery.go`.
- No OpenTelemetry tracing yet (see `docs/decisions/ADR-012-opentelemetry.md`
  for the intended direction).

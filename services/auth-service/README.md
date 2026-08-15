# CodeShop Auth Service

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![gRPC](https://img.shields.io/badge/gRPC-v1.70-244c5a?logo=google)](https://grpc.io)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-Database-4169E1?logo=postgresql&logoColor=white)](https://postgresql.org)
[![Redis](https://img.shields.io/badge/Redis-Cache-DC382D?logo=redis&logoColor=white)](https://redis.io)

Authentication and authorization microservice for the **CodeShop** digital marketplace.

The Auth Service manages user identity, registration, login, Argon2id password hashing, JWT access token generation, refresh token rotation, token validation, and instant revocation.

---

## 1. Overview & Responsibilities

### What Auth Service owns:
- User registration and account management
- Password hashing using Argon2id
- User authentication and credential verification
- Access token issuance (JWT)
- Refresh token session tracking and token rotation
- Token validation (stateless + stateful revocation checks)
- Logout / session revocation
- User identity retrieval (`GetUser`)
- Role assignment (`buyer`, `admin`)

### What Auth Service does NOT own:
- Product catalog or pricing
- Shopping cart or checkout
- Order management
- Payment processing or webhooks
- Digital downloads or signed URLs
- Notification sending

---

## 2. Architecture & Tech Stack

```text
                     Client / API Gateway
                              │
                              ▼
                      ┌───────────────┐
                      │  Auth Service │ (gRPC :50051)
                      └───────┬───────┘
                              │
               ┌──────────────┴──────────────┐
               ▼                             ▼
         PostgreSQL                        Redis
     (Users & Sessions)              (Revocation & Cache)
```

- **Language**: Go 1.22+
- **Transport**: gRPC / Protocol Buffers (`codeshop.auth.v1`)
- **Password Security**: Argon2id
- **Access Tokens**: JWT (HMAC-SHA256)
- **Database**: PostgreSQL
- **Session Cache**: Redis
- **Containerization**: Docker (multi-stage non-root build)

---

## 3. gRPC API Specification

Service: `codeshop.auth.v1.AuthService`

### `Register`
- **Purpose**: Registers a new user account with role `buyer` by default.
- **Request**: `email`, `password`, `full_name`
- **Response**: `user_id`, `email`, `full_name`, `role`, `created_at`
- **Possible Errors**: `AlreadyExists` (email taken), `InvalidArgument` (invalid email/password format).

### `Login`
- **Purpose**: Authenticates credentials and returns a JWT token pair.
- **Request**: `email`, `password`
- **Response**: `access_token`, `refresh_token`, `expires_in`, `token_type` ("Bearer"), `user_id`
- **Possible Errors**: `Unauthenticated` (wrong email/password), `PermissionDenied` (account suspended).

### `ValidateToken`
- **Purpose**: Validates an access token and returns claims.
- **Request**: `access_token`
- **Response**: `valid` (bool), `user_id`, `email`, `role`, `expires_at`
- **Possible Errors**: Returns `valid: false` for expired or tampered tokens.

### `RefreshToken`
- **Purpose**: Rotates an old refresh token to issue a new access and refresh token pair.
- **Request**: `refresh_token`
- **Response**: `access_token`, `refresh_token`, `expires_in`, `token_type`
- **Possible Errors**: `Unauthenticated` (invalid, expired, or revoked refresh token).

### `Logout`
- **Purpose**: Revokes the given refresh token or all user sessions.
- **Request**: `refresh_token`, `user_id`
- **Response**: `success` (bool), `message`
- **Possible Errors**: `Internal`.

### `GetUser`
- **Purpose**: Fetches public profile metadata of a user.
- **Request**: `user_id`
- **Response**: `user_id`, `email`, `full_name`, `role`, `status`, `created_at`, `updated_at`
- **Possible Errors**: `NotFound` (user not found), `InvalidArgument` (malformed UUID).

---

## 4. Environment Variables

| Variable | Default Value | Description |
|---|---|---|
| `ENVIRONMENT` | `development` | Environment mode (`development`, `staging`, `production`) |
| `LOG_LEVEL` | `info` | Logging verbosity |
| `GRPC_PORT` | `50051` | gRPC server listening port |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/codeshop_auth?sslmode=disable` | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6379/0` | Redis connection string |
| `JWT_SECRET` | *(required in prod)* | HMAC signing key for JWTs |
| `JWT_ISSUER` | `codeshop-auth-service` | Issuer claim in JWTs |
| `ACCESS_TOKEN_EXPIRATION` | `15m` | Lifetime of access tokens |
| `REFRESH_TOKEN_EXPIRATION` | `168h` | Lifetime of refresh tokens (7 days) |

---

## 5. Database & Migrations

The service uses PostgreSQL with `uuid-ossp` for primary key generation.

### Tables:
- `users`: Stores user identity, Argon2id password hashes, roles, and status.
- `refresh_tokens`: Stores hashed refresh token sessions for revocation auditing.

### Migration Commands:
```bash
# Run migrations up
make migrate-up

# Rollback 1 migration
make migrate-down
```

---

## 6. Local Development & Testing

### Run locally:
```bash
# Install dependencies & generate protobuf
make proto

# Run server
make run
```

### Run tests:
```bash
# Run unit & integration tests
make test

# Run tests with race condition detector
make test-race
```

### Build Docker image:
```bash
make docker-build
```

---

## 7. Troubleshooting

- **`pq: relation "users" does not exist`**: Ensure SQL migrations (`migrations/001_create_users.up.sql`) have been executed against your PostgreSQL database.
- **`connection refused to Redis`**: The Auth Service will log a warning and fall back to PostgreSQL database checks for token revocation if Redis is temporarily unreachable.
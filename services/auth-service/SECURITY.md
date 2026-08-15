# Security Policy — Auth Service

The Auth Service manages user credentials, access tokens, refresh tokens, and identity authorization. Security is designed into every layer of this service.

## 1. Password Hashing
- Passwords are hashed using **Argon2id** (`golang.org/x/crypto/argon2`).
- Parameters: `Memory=64MB`, `Iterations=3`, `Parallelism=2`, `SaltLength=16B`, `KeyLength=32B`.
- Passwords are NEVER logged, returned in API responses, or stored in plaintext.

## 2. JWT Access Tokens
- Signed using **HMAC-SHA256** with a configurable secret key (`JWT_SECRET`).
- Short time-to-live (`ACCESS_TOKEN_EXPIRATION`, default 15 minutes).
- Claims include `sub` (User ID), `email`, `role`, `token_type`, `jti`, `exp`, `iat`, `iss`.

## 3. Refresh Tokens & Rotation
- Refresh tokens have longer TTL (default 7 days).
- Token Rotation: Each use of a refresh token invalidates the old token and issues a new token pair.
- Instant Revocation: Token hashes (`SHA256`) are stored in Redis/PostgreSQL allowing real-time logout and session invalidation.

## 4. Logging Security
- The gRPC logger interceptor explicitly avoids logging request payloads that contain passwords or tokens.
- Internal errors are mapped to generic gRPC status codes (`codes.Unauthenticated`, `codes.AlreadyExists`) without leaking database constraints or stack traces.

## 5. Input Validation & Prevention
- Email validation uses RFC 5322 parsing.
- Password policy enforces a minimum of 8 characters.
- UUID parameter validation prevents SQL injection risks.

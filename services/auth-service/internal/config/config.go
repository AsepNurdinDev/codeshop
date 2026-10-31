// Package config loads Auth Service configuration from environment
// variables (12-factor style), so the same binary/image can run locally,
// in Docker Compose, and later in Kubernetes without code changes.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the Auth Service.
type Config struct {
	// App
	Env         string // "development", "production", etc.
	ServiceName string

	// HTTP (public REST, consumed directly today and via API Gateway later
	// - see docs/architecture/SERVICE_CATALOG.md §2)
	HTTPPort string

	// gRPC (internal, ValidateToken/GetUser - see proto/auth.proto)
	GRPCPort string

	// PostgreSQL (Auth DB - docs/architecture/DATABASE_DESIGN.md §2)
	DatabaseURL         string
	DatabaseMaxConns    int32
	DatabaseConnTimeout time.Duration

	// Redis (refresh token sessions & login rate limiting -
	// docs/decisions/ADR-007-redis.md)
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// RabbitMQ (publishes UserRegistered - docs/architecture/EVENT_DESIGN.md)
	RabbitMQURL      string
	RabbitMQExchange string

	// JWT / security
	JWTSecret          string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	PasswordHashParams Argon2Params

	// Rate limiting (login brute-force protection -
	// docs/architecture/SERVICE_CATALOG.md §2)
	LoginRateLimitAttempts int
	LoginRateLimitWindow   time.Duration
}

// Argon2Params configures Argon2id password hashing
// (docs/architecture/DATABASE_DESIGN.md §2: "password_hash - hasil Argon2id").
type Argon2Params struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultArgon2Params returns OWASP-recommended baseline parameters for
// Argon2id (19 MiB memory, 2 iterations, 1 degree of parallelism is the
// OWASP minimum; we use a slightly stronger default suitable for a
// server-side login path).
func DefaultArgon2Params() Argon2Params {
	return Argon2Params{
		Memory:      64 * 1024, // 64 MiB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// Load reads configuration from environment variables, applying sane
// development defaults where possible. It fails fast (returns an error)
// for anything security-critical that has no safe default, such as
// JWT_SECRET.
func Load() (*Config, error) {
	cfg := &Config{
		Env:         getEnv("APP_ENV", "development"),
		ServiceName: getEnv("SERVICE_NAME", "auth-service"),

		HTTPPort: getEnv("HTTP_PORT", "8080"),
		GRPCPort: getEnv("GRPC_PORT", "9090"),

		DatabaseURL:         getEnv("DATABASE_URL", "postgres://auth:auth@localhost:5432/auth_db?sslmode=disable"),
		DatabaseMaxConns:    int32(getEnvInt("DATABASE_MAX_CONNS", 10)),
		DatabaseConnTimeout: getEnvDuration("DATABASE_CONN_TIMEOUT", 5*time.Second),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		RabbitMQURL:      getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitMQExchange: getEnv("RABBITMQ_EXCHANGE", "codeshop.events"),

		JWTSecret:       getEnv("JWT_SECRET", ""),
		AccessTokenTTL:  getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: getEnvDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour),

		LoginRateLimitAttempts: getEnvInt("LOGIN_RATE_LIMIT_ATTEMPTS", 5),
		LoginRateLimitWindow:   getEnvDuration("LOGIN_RATE_LIMIT_WINDOW", 15*time.Minute),
	}
	cfg.PasswordHashParams = DefaultArgon2Params()

	if cfg.JWTSecret == "" {
		if cfg.Env == "production" {
			return nil, fmt.Errorf("JWT_SECRET must be set in production")
		}
		// Development-only fallback so `go run` works out of the box.
		cfg.JWTSecret = "dev-only-insecure-secret-change-me"
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

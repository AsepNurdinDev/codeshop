package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment            string
	LogLevel               string
	ServerPort             string
	GRPCPort               string
	DatabaseURL            string
	RedisURL               string
	JWTSecret              string
	JWTIssuer              string
	AccessTokenExpiration  time.Duration
	RefreshTokenExpiration time.Duration
}

func LoadConfig() *Config {
	return &Config{
		Environment:            getEnv("ENVIRONMENT", "development"),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		ServerPort:             getEnv("SERVER_PORT", "8081"),
		GRPCPort:               getEnv("GRPC_PORT", "50051"),
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/codeshop_auth?sslmode=disable"),
		RedisURL:               getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:              getEnv("JWT_SECRET", "super-secret-jwt-key-codeshop-production-auth"),
		JWTIssuer:              getEnv("JWT_ISSUER", "codeshop-auth-service"),
		AccessTokenExpiration:  getEnvDuration("ACCESS_TOKEN_EXPIRATION", 15*time.Minute),
		RefreshTokenExpiration: getEnvDuration("REFRESH_TOKEN_EXPIRATION", 7*24*time.Hour),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	valStr := getEnv(key, "")
	if valStr == "" {
		return fallback
	}

	// Try parsing duration string like "15m", "7h"
	if dur, err := time.ParseDuration(valStr); err == nil {
		return dur
	}

	// Try parsing raw integer seconds
	if sec, err := strconv.Atoi(valStr); err == nil {
		return time.Duration(sec) * time.Second
	}

	return fallback
}

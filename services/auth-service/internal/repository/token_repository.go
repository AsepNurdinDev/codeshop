package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/asepnrdn/codeshop/services/auth-service/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type TokenRepository interface {
	CreateSession(ctx context.Context, session *domain.RefreshTokenSession) error
	GetSessionByHash(ctx context.Context, tokenHash string) (*domain.RefreshTokenSession, error)
	RevokeSession(ctx context.Context, tokenHash string) error
	RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error
}

type tokenRepository struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewTokenRepository(db *sql.DB, rdb *redis.Client) TokenRepository {
	return &tokenRepository{
		db:  db,
		rdb: rdb,
	}
}

func (r *tokenRepository) CreateSession(ctx context.Context, session *domain.RefreshTokenSession) error {
	// 1. Persist to PostgreSQL
	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		session.ID,
		session.UserID,
		session.TokenHash,
		session.ExpiresAt,
		session.Revoked,
		session.CreatedAt,
		session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("db error saving refresh token session: %w", err)
	}

	// 2. Cache session in Redis if available
	if r.rdb != nil {
		ttl := time.Until(session.ExpiresAt)
		if ttl > 0 {
			redisKey := fmt.Sprintf("session:refresh:%s", session.TokenHash)
			_ = r.rdb.Set(ctx, redisKey, session.UserID.String(), ttl).Err()
		}
	}

	return nil
}

func (r *tokenRepository) GetSessionByHash(ctx context.Context, tokenHash string) (*domain.RefreshTokenSession, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at, updated_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`
	row := r.db.QueryRowContext(ctx, query, tokenHash)

	var s domain.RefreshTokenSession
	err := row.Scan(
		&s.ID,
		&s.UserID,
		&s.TokenHash,
		&s.ExpiresAt,
		&s.Revoked,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInvalidToken
		}
		return nil, fmt.Errorf("db error getting refresh token: %w", err)
	}

	if s.Revoked {
		return nil, domain.ErrTokenRevoked
	}

	if time.Now().After(s.ExpiresAt) {
		return nil, domain.ErrExpiredToken
	}

	return &s, nil
}

func (r *tokenRepository) RevokeSession(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE token_hash = $1
	`
	_, err := r.db.ExecContext(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("db error revoking refresh token: %w", err)
	}

	if r.rdb != nil {
		redisKey := fmt.Sprintf("session:refresh:%s", tokenHash)
		_ = r.rdb.Del(ctx, redisKey).Err()
	}

	return nil
}

func (r *tokenRepository) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND revoked = FALSE
	`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("db error revoking user sessions: %w", err)
	}

	return nil
}

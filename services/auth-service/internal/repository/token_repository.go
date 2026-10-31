package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/codeshop-platform/auth-service/internal/domain"
)

const redisRefreshTokenKeyPrefix = "auth:refresh_token:"

// TokenRepository manages refresh-token sessions.
//
// Design (see migrations/002_create_refresh_tokens.up.sql and
// docs/decisions/ADR-007-redis.md):
//   - Redis is the source of truth for whether a refresh token is
//     *currently* valid: it holds the session with a TTL matching the
//     token's expiry, so an expired token simply falls out of Redis, and
//     a revoked/logged-out token is deleted immediately (O(1) revocation).
//   - PostgreSQL keeps a durable, queryable record of every token ever
//     issued for audit trail and refresh-token-reuse detection (a token
//     that is rotated but presented again is a strong signal of theft).
type TokenRepository interface {
	// Store persists a newly issued refresh token in both Redis (for fast
	// validity checks) and Postgres (for the audit trail).
	Store(ctx context.Context, t *domain.RefreshToken) error

	// Get looks up a refresh token's current validity from Redis (the hot
	// path used on every /auth/refresh call).
	Get(ctx context.Context, tokenID uuid.UUID) (*domain.RefreshToken, error)

	// Rotate marks the old token as replaced by the new token's ID (both
	// stores), and stores the new token. Used on every successful refresh
	// to implement refresh-token rotation
	// (docs/architecture/SERVICE_CATALOG.md §2).
	Rotate(ctx context.Context, oldTokenID uuid.UUID, newToken *domain.RefreshToken) error

	// Revoke immediately invalidates a token (logout, or reuse-detected).
	Revoke(ctx context.Context, tokenID uuid.UUID) error

	// RevokeAllForUser invalidates every refresh token belonging to a
	// user, e.g. on password change or "log out of all devices".
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type tokenRepository struct {
	redis *redis.Client
	pool  *pgxpool.Pool
}

// NewTokenRepository returns a Redis+Postgres backed TokenRepository.
func NewTokenRepository(redisClient *redis.Client, pool *pgxpool.Pool) TokenRepository {
	return &tokenRepository{redis: redisClient, pool: pool}
}

func (r *tokenRepository) Store(ctx context.Context, t *domain.RefreshToken) error {
	if err := r.writeRedisSession(ctx, t); err != nil {
		return err
	}

	const q = `
		INSERT INTO refresh_tokens (id, user_id, issued_at, expires_at, revoked)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := r.pool.Exec(ctx, q, t.TokenID, t.UserID, t.IssuedAt, t.ExpiresAt, t.Revoked); err != nil {
		return fmt.Errorf("persist refresh token audit record: %w", err)
	}
	return nil
}

func (r *tokenRepository) Get(ctx context.Context, tokenID uuid.UUID) (*domain.RefreshToken, error) {
	key := redisRefreshTokenKeyPrefix + tokenID.String()
	vals, err := r.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("get refresh token session: %w", err)
	}
	if len(vals) == 0 {
		return nil, domain.ErrRefreshTokenNotFound
	}

	userID, err := uuid.Parse(vals["user_id"])
	if err != nil {
		return nil, fmt.Errorf("parse user_id from session: %w", err)
	}
	issuedAt, _ := time.Parse(time.RFC3339, vals["issued_at"])
	expiresAt, _ := time.Parse(time.RFC3339, vals["expires_at"])

	return &domain.RefreshToken{
		TokenID:   tokenID,
		UserID:    userID,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
		Revoked:   vals["revoked"] == "true",
	}, nil
}

func (r *tokenRepository) Rotate(ctx context.Context, oldTokenID uuid.UUID, newToken *domain.RefreshToken) error {
	// Invalidate the old session immediately so it cannot be replayed.
	if err := r.redis.Del(ctx, redisRefreshTokenKeyPrefix+oldTokenID.String()).Err(); err != nil {
		return fmt.Errorf("delete old refresh token session: %w", err)
	}

	if err := r.writeRedisSession(ctx, newToken); err != nil {
		return err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin rotate tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`UPDATE refresh_tokens SET revoked = TRUE, revoked_at = now(), replaced_by = $2 WHERE id = $1`,
		oldTokenID, newToken.TokenID,
	); err != nil {
		return fmt.Errorf("mark old refresh token replaced: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, issued_at, expires_at, revoked) VALUES ($1, $2, $3, $4, $5)`,
		newToken.TokenID, newToken.UserID, newToken.IssuedAt, newToken.ExpiresAt, newToken.Revoked,
	); err != nil {
		return fmt.Errorf("persist rotated refresh token: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *tokenRepository) Revoke(ctx context.Context, tokenID uuid.UUID) error {
	if err := r.redis.Del(ctx, redisRefreshTokenKeyPrefix+tokenID.String()).Err(); err != nil {
		return fmt.Errorf("delete refresh token session: %w", err)
	}

	tag, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked = TRUE, revoked_at = now() WHERE id = $1 AND revoked = FALSE`,
		tokenID,
	)
	if err != nil {
		return fmt.Errorf("mark refresh token revoked: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Not necessarily an error: token may already be revoked/rotated,
		// or may never have existed. Logout should still be idempotent.
		return nil
	}
	return nil
}

func (r *tokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	rows, err := r.pool.Query(ctx,
		`SELECT id FROM refresh_tokens WHERE user_id = $1 AND revoked = FALSE`, userID,
	)
	if err != nil {
		return fmt.Errorf("list active refresh tokens: %w", err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan refresh token id: %w", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate refresh tokens: %w", err)
	}

	pipe := r.redis.Pipeline()
	for _, id := range ids {
		pipe.Del(ctx, redisRefreshTokenKeyPrefix+id.String())
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("bulk delete refresh token sessions: %w", err)
	}

	if _, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked = TRUE, revoked_at = now() WHERE user_id = $1 AND revoked = FALSE`,
		userID,
	); err != nil {
		return fmt.Errorf("mark all refresh tokens revoked: %w", err)
	}
	return nil
}

func (r *tokenRepository) writeRedisSession(ctx context.Context, t *domain.RefreshToken) error {
	key := redisRefreshTokenKeyPrefix + t.TokenID.String()
	ttl := time.Until(t.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("refresh token already expired at store time")
	}

	pipe := r.redis.TxPipeline()
	pipe.HSet(ctx, key, map[string]any{
		"user_id":    t.UserID.String(),
		"issued_at":  t.IssuedAt.Format(time.RFC3339),
		"expires_at": t.ExpiresAt.Format(time.RFC3339),
		"revoked":    "false",
	})
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("write refresh token session: %w", err)
	}
	return nil
}

package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codeshop-platform/auth-service/internal/domain"
)

// UserRepository persists the User aggregate in the Auth DB (PostgreSQL,
// docs/decisions/ADR-006-postgresql-database.md). Auth Service is the
// exclusive owner/writer of this table (docs/architecture/DATABASE_DESIGN.md §1).
type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Update(ctx context.Context, u *domain.User) error
}

type postgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository returns a PostgreSQL-backed UserRepository.
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &postgresUserRepository{pool: pool}
}

func (r *postgresUserRepository) Create(ctx context.Context, u *domain.User) error {
	const q = `
		INSERT INTO users (id, email, password_hash, full_name, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, q,
		u.ID, strings.ToLower(u.Email), u.PasswordHash, u.FullName,
		string(u.Role), string(u.Status), u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailAlreadyInUse
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `
		SELECT id, email, password_hash, full_name, role, status, created_at, updated_at
		FROM users WHERE id = $1
	`
	return scanUser(r.pool.QueryRow(ctx, q, id))
}

func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `
		SELECT id, email, password_hash, full_name, role, status, created_at, updated_at
		FROM users WHERE email = LOWER($1)
	`
	return scanUser(r.pool.QueryRow(ctx, q, email))
}

func (r *postgresUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM users WHERE email = LOWER($1))`
	var exists bool
	if err := r.pool.QueryRow(ctx, q, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("check email existence: %w", err)
	}
	return exists, nil
}

func (r *postgresUserRepository) Update(ctx context.Context, u *domain.User) error {
	const q = `
		UPDATE users
		SET email = $2, password_hash = $3, full_name = $4, role = $5, status = $6, updated_at = $7
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, q,
		u.ID, strings.ToLower(u.Email), u.PasswordHash, u.FullName,
		string(u.Role), string(u.Status), u.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailAlreadyInUse
		}
		return fmt.Errorf("update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// rowScanner abstracts pgx.Row so scanUser works for both QueryRow and,
// if ever needed, a row from a batch/transaction.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*domain.User, error) {
	var (
		u      domain.User
		role   string
		status string
	)
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &role, &status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.Role = domain.Role(role)
	u.Status = domain.UserStatus(status)
	return &u, nil
}

// isUniqueViolation detects a Postgres unique_violation (SQLSTATE 23505),
// e.g. a duplicate email on INSERT/UPDATE.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

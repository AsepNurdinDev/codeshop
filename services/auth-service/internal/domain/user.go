package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// UserStatus reflects account state without deleting the underlying row,
// per docs/architecture/DATABASE_DESIGN.md §2 ("status - untuk mendukung
// suspend user tanpa hapus data").
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
)

// Domain-level sentinel errors. Handlers/services map these to the
// standard error envelope described in docs/architecture/API_DESIGN.md §6.
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailAlreadyInUse = errors.New("email already in use")
	ErrInvalidCredential = errors.New("invalid email or password")
	ErrAccountSuspended  = errors.New("account is suspended")
	ErrWeakPassword      = errors.New("password does not meet minimum strength requirements")
	ErrInvalidEmail      = errors.New("email is not valid")
)

// User is the Auth Service's core aggregate. Auth Service is the sole
// owner of this entity (docs/architecture/DATABASE_DESIGN.md §1); other
// services only ever hold a user_id reference, never a foreign key into
// this table.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	FullName     string
	Role         Role
	Status       UserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewUser builds a new buyer account in the active state. Password hashing
// happens in the service layer (domain stays persistence/crypto agnostic).
func NewUser(email, passwordHash, fullName string) *User {
	now := time.Now().UTC()
	return &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		FullName:     fullName,
		Role:         RoleBuyer,
		Status:       UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// IsActive reports whether the account is allowed to authenticate.
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

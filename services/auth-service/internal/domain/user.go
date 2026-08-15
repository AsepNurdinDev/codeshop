package domain

import (
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleBuyer Role = "buyer"
	RoleAdmin Role = "admin"
)

func (r Role) IsValid() bool {
	return r == RoleBuyer || r == RoleAdmin
}

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
)

func (s UserStatus) IsValid() bool {
	return s == UserStatusActive || s == UserStatusSuspended
}

type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	FullName     string     `json:"full_name"`
	Role         Role       `json:"role"`
	Status       UserStatus `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return errors.New("email is required")
	}
	_, err := mail.ParseAddress(email)
	if err != nil {
		return errors.New("invalid email format")
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	return nil
}

func ValidateFullName(fullName string) error {
	if strings.TrimSpace(fullName) == "" {
		return errors.New("full name is required")
	}
	return nil
}

package domain

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrEmailAlreadyExists  = errors.New("email already registered")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrAccountSuspended    = errors.New("account is suspended")
	ErrInvalidToken        = errors.New("invalid or malformed token")
	ErrExpiredToken        = errors.New("token has expired")
	ErrTokenRevoked        = errors.New("token has been revoked")
	ErrInvalidTokenType    = errors.New("invalid token type")
	ErrUnauthorized        = errors.New("unauthorized request")
	ErrInternalServerError = errors.New("internal server error")
)

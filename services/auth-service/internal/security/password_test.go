package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashPasswordAndCompare(t *testing.T) {
	password := "SecretP@ssw0rd123"

	// 1. Hash password
	hash, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Contains(t, hash, "$argon2id$")

	// 2. Compare valid password
	match, err := ComparePassword(password, hash)
	assert.NoError(t, err)
	assert.True(t, match)

	// 3. Compare wrong password
	matchWrong, err := ComparePassword("WrongPassword!", hash)
	assert.NoError(t, err)
	assert.False(t, matchWrong)
}

func TestHashPasswordEmpty(t *testing.T) {
	_, err := HashPassword("")
	assert.Error(t, err)
}

func TestCompareInvalidHashFormat(t *testing.T) {
	match, err := ComparePassword("password", "invalid-hash-string")
	assert.Error(t, err)
	assert.False(t, match)
}

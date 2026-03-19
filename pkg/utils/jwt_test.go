package utils

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndParseJWT(t *testing.T) {
	secret := "test-secret"
	userID := uuid.New()

	token, err := GenerateJWT(userID, secret, 72)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	parsed, err := ParseJWT(token, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, parsed)
}

func TestParseJWT_InvalidToken(t *testing.T) {
	_, err := ParseJWT("invalid-token", "secret")
	assert.Error(t, err)
}

func TestParseJWT_WrongSecret(t *testing.T) {
	userID := uuid.New()
	token, _ := GenerateJWT(userID, "secret1", 72)
	_, err := ParseJWT(token, "secret2")
	assert.Error(t, err)
}

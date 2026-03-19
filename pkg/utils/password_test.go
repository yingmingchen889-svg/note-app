package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndCheckPassword(t *testing.T) {
	password := "mypassword123"
	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEqual(t, password, hash)
	assert.True(t, CheckPassword(password, hash))
}

func TestCheckPassword_Wrong(t *testing.T) {
	hash, _ := HashPassword("correct")
	assert.False(t, CheckPassword("wrong", hash))
}

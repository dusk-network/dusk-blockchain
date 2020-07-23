package server_test

import (
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestCreateDropSession(t *testing.T) {
	assert := assert.New(t)
	jwt, err := authClient.CreateSession()
	assert.NoError(err)
	assert.NotEmpty(jwt)

	// first time we drop the session there should be no error
	assert.NoError(authClient.DropSession())
	// if we drop the session immediately after, we should get an
	// authorization error since the session has been dropped and we did not
	// recreate it
	assert.Error(authClient.DropSession())
}

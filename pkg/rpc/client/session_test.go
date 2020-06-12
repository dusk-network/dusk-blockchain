package client_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDropSession(t *testing.T) {
	assert := assert.New(t)
	_, err := nodeClient.GetSessionConn()
	assert.NoError(err)

	// first time we drop the session there should be no error
	assert.NoError(nodeClient.DropSession())
	// if we drop the session immediately after, we should get an
	// authorization error since the session has been dropped and we did not
	// recreate it
	assert.Error(nodeClient.DropSession())
}

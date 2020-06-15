package client_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestCreateDropSession(t *testing.T) {
	assert := assert.New(t)
	_, err := nodeClient.GetSessionConn(grpc.WithInsecure(), grpc.WithBlock())
	assert.NoError(err)

	// first time we drop the session there should be no error
	assert.NoError(nodeClient.DropSession(grpc.WithInsecure()))
	// if we drop the session immediately after, we should get an
	// authorization error since the session has been dropped and we did not
	// recreate it
	assert.Error(nodeClient.DropSession(grpc.WithInsecure()))
}

// TestPersistentSession tests that subsequent calls to
// NodeClient.GetSessionConn do not recreate a session
func TestPersistentSession(t *testing.T) {
	assert := assert.New(t)
	token1, err := nodeClient.GetSessionConn(grpc.WithInsecure(), grpc.WithBlock())
	assert.NoError(err)

	token2, err := nodeClient.GetSessionConn(grpc.WithInsecure(), grpc.WithBlock())
	assert.NoError(err)
	assert.Equal(token1, token2)
}

// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
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

// TestPersistentSession tests that the client can exploit the session injected
// through the AuthClientInterceptor and perform authenticated calls.
func TestPersistentSession(t *testing.T) {
	assert := assert.New(t)
	conn, err := nodeClient.GetSessionConn(grpc.WithInsecure(), grpc.WithBlock())
	assert.NoError(err)
	assert.NoError(createDumbWallet(conn))
	assert.NoError(loadDumbWallet(conn))

	nodeClient.DropSession(grpc.WithInsecure(), grpc.WithBlock())
	assert.Error(createDumbWallet(conn))
}

func createDumbWallet(conn *grpc.ClientConn) error {
	// create a wallet client
	walletClient := node.NewWalletClient(conn)

	// spawn an authenticated RPC call
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	in := &node.CreateRequest{}

	// the mock always returns success, so if there is an error it is likely to
	// be linked to the session and authorization layer
	_, err := walletClient.CreateWallet(ctx, in)
	return err
}

func loadDumbWallet(conn *grpc.ClientConn) error {
	// create a wallet client
	walletClient := node.NewWalletClient(conn)

	// spawn an authenticated RPC call
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// the mock always returns success, so if there is an error it is likely to
	// be linked to the session and authorization layer
	_, err := walletClient.LoadWallet(ctx, &node.LoadRequest{})
	return err
}

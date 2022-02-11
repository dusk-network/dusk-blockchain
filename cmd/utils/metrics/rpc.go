// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package metrics

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

// Client is a wrapper for a gRPC client. It establishes connection with
// the server on startup, and then handles requests from other components
// over the RPCBus.
type Client struct {
	// rusk.StateClient
	// node.WalletClient
	// node.TransactorClient
	conn *grpc.ClientConn
}

// InitRPCClients opens the connection with the Rusk gRPC server, and
// initializes the different clients which can speak to the Rusk server.
//
// As the Rusk server is a fundamental part of the node functionality,
// this function will panic if the connection can not be established
// successfully.
func InitRPCClients(ctx context.Context, address string) *Client {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithAuthority("dummy"))
	if err != nil {
		log.Panic(err)
	}

	return &Client{
		// StateClient:       rusk.NewStateClient(conn),
		// WalletClient:     node.NewWalletClient(conn),
		// TransactorClient: node.NewTransactorClient(conn),
		conn: conn,
	}
}

// Close the connection to the gRPC server.
func (c *Client) Close() error {
	return c.conn.Close()
}

// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package grpcclient

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// grpcClient is a helper for establishing grpc client connection reusing
// default RPCConfiguration struct. TODO: It seems similar impl already used in
// wallet pkg and test harness. Refactor this by providing a single client.
type grpcClient struct {
	dialTimeout int64
	conn        *grpc.ClientConn
}

// TryConnect initializes a grpcClient to dusk-blockchain node grpc interface.
// For over-tcp communication, it could enable TLS and Basic Authentication.
func (c *grpcClient) TryConnect(addr string) error {
	dialOptions := make([]grpc.DialOption, 0)
	dialOptions = append(dialOptions, grpc.WithBlock())
	dialOptions = append(dialOptions, grpc.WithInsecure())

	// Init dial timeout
	var dialCtx context.Context

	if c.dialTimeout > 0 {
		var cancel context.CancelFunc

		dialCtx, cancel = context.WithTimeout(context.Background(),
			time.Duration(c.dialTimeout)*time.Second)
		defer cancel()
	}

	// Set up a connection to the server.
	conn, err := grpc.DialContext(dialCtx, addr, dialOptions...)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

// Close conn.
func (c *grpcClient) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

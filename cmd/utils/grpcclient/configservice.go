// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package grpcclient

import (
	"context"
	"time"

	node "github.com/dusk-network/dusk-protobuf/autogen/go/node"
)

// SetConfigReq describes a set-config request.
type SetConfigReq struct {
	Address string

	Name     string
	NewValue string
}

// TrySetConfig implement a cli client over node.ConfigClient grpc interface.
func TrySetConfig(r SetConfigReq) error {
	var err error

	c := grpcClient{dialTimeout: 5}
	if err = c.TryConnect(r.Address); err != nil {
		return err
	}

	defer c.Close()

	client := node.NewConfigClient(c.conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if r.Name == "logger.level" {
		_, err = client.ChangeLogLevel(ctx, &node.LogLevelRequest{Level: r.NewValue})
	}

	return err
}

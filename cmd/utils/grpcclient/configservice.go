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

	"google.golang.org/grpc"
)

// SetConfigReq describes a set-config request.
type SetConfigReq struct {
	Host string

	Name     string
	NewValue string
}

// SetConfig implement a cli client over node.ConfigClient grpc interface.
func SetConfig(r SetConfigReq) error {
	// TODO: enable tcp transport
	addr := "unix://" + r.Host

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	defer func() {
		_ = conn.Close()
	}()

	client := node.NewConfigClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if r.Name == "logger.level" {
		_, err = client.ChangeLogLevel(ctx, &node.LogLevelRequest{Level: r.NewValue})
	}

	return err
}

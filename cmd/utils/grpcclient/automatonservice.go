// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package grpcclient

import (
	"context"
	"strings"
	"time"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
)

// AutomateStakes will enable the use of the stake automaton in a node.
func AutomateStakes(address string, sendStakeTimeout int) error {
	// Add UNIX prefix in case we're using unix sockets.
	if strings.Contains(address, ".sock") {
		address = "unix://" + address
	}

	c := grpcClient{dialTimeout: 5}
	if err := c.TryConnect(address); err != nil {
		return err
	}

	defer c.Close()

	stakeClient := node.NewProvisionerClient(c.conn)

	return automateStakes(stakeClient, sendStakeTimeout)
}

func automateStakes(c node.ProvisionerClient, sendBidTimeout int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sendBidTimeout)*time.Second)
	defer cancel()

	_, err := c.AutomateStakes(ctx, &node.EmptyRequest{})
	return err
}

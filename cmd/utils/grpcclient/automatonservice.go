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

// AutomateStakesAndBids will enable the use of the stake and bid automaton in a node.
func AutomateStakesAndBids(address string, sendStakeTimeout, sendBidTimeout int) error {
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
	bidClient := node.NewBlockGeneratorClient(c.conn)

	if err := automateStakes(stakeClient, sendStakeTimeout); err != nil {
		return err
	}

	return automateBids(bidClient, sendBidTimeout)
}

func automateStakes(c node.ProvisionerClient, sendBidTimeout int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sendBidTimeout)*time.Second)
	defer cancel()

	_, err := c.AutomateStakes(ctx, &node.EmptyRequest{})
	return err
}

func automateBids(c node.BlockGeneratorClient, sendStakeTimeout int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sendStakeTimeout)*time.Second)
	defer cancel()

	_, err := c.AutomateBids(ctx, &node.EmptyRequest{})
	return err
}

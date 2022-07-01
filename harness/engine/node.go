// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package engine

import (
	"strconv"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/client"
	"github.com/machinebox/graphql"
)

// DuskNode is the struct representing a node instance in the local Network.
type DuskNode struct {
	Id              string //nolint
	ConfigProfileID string

	// Fields represents a dusk-blockchain instance.
	Cfg config.Registry
	Gql *graphql.Client

	GRPCClient *client.NodeClient

	// Dusk-blockchain node directory.
	Dir string
}

// NewDuskNode instantiates a new DuskNode.
func NewDuskNode(graphqlPort, nodeID int, profileID string, requireSession bool) *DuskNode {
	node := new(DuskNode)
	node.Id = strconv.Itoa(nodeID)
	node.ConfigProfileID = profileID

	node.Cfg = config.Registry{}
	node.Cfg.General.Network = "devnet"
	node.Cfg.Network.ServiceFlag = 1
	node.Cfg.Gql.Address = "127.0.0.1:" + strconv.Itoa(graphqlPort)
	node.Cfg.Gql.Network = "tcp" //nolint

	node.Cfg.RPC.RequireSession = requireSession
	node.Cfg.RPC.SessionDurationMins = 5

	if *RPCNetworkType == "unix" { //nolint
		node.Cfg.RPC.Network = "unix"
		node.Cfg.RPC.Address = "/dusk-grpc.sock"
	} else {
		node.Cfg.RPC.Network = "tcp" //nolint
		node.Cfg.RPC.Address = "127.0.0.1:" + node.Id
	}

	/*
		node.Cfg.RPC.Rusk.Network = "tcp" //nolint
		node.Cfg.RPC.Rusk.Address = "127.0.0.1:" + strconv.Itoa(nodeID+1000)
	*/
	node.Cfg.RPC.Rusk.Network = "unix" //nolint
	node.Cfg.RPC.Rusk.Address = "/rusk-grpc.sock"

	node.Cfg.RPC.Rusk.ContractTimeout = 20000
	node.Cfg.RPC.Rusk.DefaultTimeout = 1000
	node.Cfg.RPC.Rusk.ConnectionTimeout = 10000

	node.Gql = graphql.NewClient("http://" + node.Cfg.Gql.Address)
	return node
}

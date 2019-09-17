package engine

import (
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/machinebox/graphql"
)

type DuskNode struct {
	Id              string
	ConfigProfileID string

	// fields represents a dusk-blockchain instance
	Cfg config.Registry
	Gql *graphql.Client

	// dusk-blockchain node directory
	Dir string
}

func NewDuskNode(gqlPort, nodeID string, profileID string) *DuskNode {

	node := new(DuskNode)
	node.Id = nodeID
	node.ConfigProfileID = profileID

	node.Cfg = config.Registry{}
	node.Cfg.Gql.Port = gqlPort

	if *RPCNetworkType == "unix" {
		node.Cfg.RPC.Network = "unix"
		node.Cfg.RPC.Address = "/tmp/dusk-node-" + nodeID + ".sock"
	} else {
		node.Cfg.RPC.Network = "tcp"
		node.Cfg.RPC.Address = "127.0.0.1:" + nodeID
	}

	node.Gql = graphql.NewClient("http://127.0.0.1:" + node.Cfg.Gql.Port)
	return node
}

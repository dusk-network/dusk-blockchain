package engine

import (
	"strconv"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/util/ruskmock"
	"github.com/machinebox/graphql"
)

// DuskNode is the struct representing a node instance in the local Network
type DuskNode struct {
	Id              string //nolint
	ConfigProfileID string

	// fields represents a dusk-blockchain instance
	Cfg config.Registry
	Gql *graphql.Client
	Srv *ruskmock.Server

	// dusk-blockchain node directory
	Dir string
}

// NewDuskNode instantiates a new DuskNode
func NewDuskNode(graphqlPort, nodeID int, profileID string) *DuskNode {

	node := new(DuskNode)
	node.Id = strconv.Itoa(nodeID)
	node.ConfigProfileID = profileID

	node.Cfg = config.Registry{}
	node.Cfg.Gql.Address = "127.0.0.1:" + strconv.Itoa(graphqlPort)
	node.Cfg.Gql.Network = "tcp"

	if *RPCNetworkType == "unix" { //nolint
		node.Cfg.RPC.Network = "unix"
		node.Cfg.RPC.Address = "/dusk-grpc.sock"
	} else {
		node.Cfg.RPC.Network = "tcp"
		node.Cfg.RPC.Address = "127.0.0.1:" + node.Id
	}

	node.Gql = graphql.NewClient("http://" + node.Cfg.Gql.Address)
	return node
}

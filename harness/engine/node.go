package engine

import (
	"strconv"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/client"
	"github.com/dusk-network/dusk-blockchain/pkg/util/ruskmock"
	"github.com/machinebox/graphql"
)

// DuskNode is the struct representing a node instance in the local Network
type DuskNode struct {
	Id              string //nolint
	ConfigProfileID string

	// fields represents a dusk-blockchain instance
	Cfg        config.Registry
	Gql        *graphql.Client
	Srv        *ruskmock.Server
	GRPCClient *client.NodeClient

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
	node.Cfg.Gql.Network = "tcp" //nolint

	if *RPCNetworkType == "unix" { //nolint
		node.Cfg.RPC.Network = "unix"
		node.Cfg.RPC.Address = "/dusk-grpc.sock" + node.Id
	} else {
		node.Cfg.RPC.Network = "tcp" //nolint
		node.Cfg.RPC.Address = "127.0.0.1:" + node.Id
	}

	node.Cfg.RPC.Rusk.Network = "tcp" //nolint
	node.Cfg.RPC.Rusk.Address = "127.0.0.1:" + strconv.Itoa(nodeID+1000)
	node.Cfg.RPC.Rusk.ContractTimeout = 6000
	node.Cfg.RPC.Rusk.DefaultTimeout = 1000

	node.Cfg.Genesis.Legacy = true

	node.Gql = graphql.NewClient("http://" + node.Cfg.Gql.Address)
	node.GRPCClient = client.New(node.Cfg.RPC.Network, node.Cfg.RPC.Address)
	return node
}

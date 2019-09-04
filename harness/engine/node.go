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

func NewDuskNode(gqlPort, rpcPort string, profileID string) *DuskNode {

	node := new(DuskNode)
	node.Id = rpcPort
	node.ConfigProfileID = profileID

	node.Cfg = config.Registry{}
	node.Cfg.Gql.Port = gqlPort
	node.Cfg.RPC.Port = rpcPort

	node.Gql = graphql.NewClient("http://127.0.0.1:" + node.Cfg.Gql.Port)
	return node
}

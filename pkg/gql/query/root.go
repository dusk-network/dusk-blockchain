package query

import (
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/graphql-go/graphql"
)

type Root struct {
	Query *graphql.Object
}

func NewRoot(rpcBus *rpcbus.RPCBus) *Root {

	m := mempool{rpcBus: rpcBus}

	root := Root{
		Query: graphql.NewObject(
			graphql.ObjectConfig{
				Name: "Query",
				Fields: graphql.Fields{
					"blocks":       blocks{}.getQuery(),
					"transactions": transactions{}.getQuery(),
					"mempool":      m.getQuery(),
				},
			},
		),
	}
	return &root
}

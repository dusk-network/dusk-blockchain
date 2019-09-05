package query

import (
	"context"
	"fmt"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/graphql-go/graphql"
)

type Root struct {
	Query *graphql.Object
}

func NewRoot(rpcBus *wire.RPCBus) *Root {

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

func Execute(query string, schema graphql.Schema, db database.DB) *graphql.Result {
	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
		Context:       context.WithValue(context.Background(), "database", db),
	})

	// Error check
	if len(result.Errors) > 0 {
		fmt.Printf("Unexpected errors inside ExecuteQuery: %v", result.Errors)
	}

	return result
}

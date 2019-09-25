package query

import (
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/graphql-go/graphql"
)

type version struct {
}

func (t version) getQuery() *graphql.Field {
	return &graphql.Field{
		Type:    graphql.String,
		Resolve: t.resolve,
	}
}

func (t version) resolve(p graphql.ResolveParams) (interface{}, error) {
	return fmt.Sprintf("%v", protocol.NodeVer.String()), nil
}

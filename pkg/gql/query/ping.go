package query

import (
	"github.com/graphql-go/graphql"
)

type ping struct {
}

func (t ping) getQuery() *graphql.Field {
	return &graphql.Field{
		Type:    graphql.String,
		Resolve: t.resolve,
	}
}

func (t ping) resolve(p graphql.ResolveParams) (interface{}, error) {
	return "pong", nil
}

package query

import (
	"strconv"
	"time"

	"github.com/graphql-go/graphql"
)

var startTime = time.Now().Unix()

type uptime struct {
}

func (t uptime) getQuery() *graphql.Field {
	return &graphql.Field{
		Type:    graphql.String,
		Resolve: t.resolve,
	}
}

func (t uptime) resolve(p graphql.ResolveParams) (interface{}, error) {
	return strconv.FormatInt(time.Now().Unix()-startTime, 10), nil
}

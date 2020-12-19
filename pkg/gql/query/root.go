// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package query

import (
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/graphql-go/graphql"
)

// Root represents the root of the graphql object
type Root struct {
	Query *graphql.Object
}

// NewRoot returns a Root with blocks, transactions and mempool setup
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

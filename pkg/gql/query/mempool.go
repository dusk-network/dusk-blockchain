// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package query

import (
	"bytes"
	"encoding/hex"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"

	txs "github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/graphql-go/graphql"
)

type mempool struct {
	rpcBus *rpcbus.RPCBus
}

func (t mempool) getQuery() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(Transaction),
		Args: graphql.FieldConfigArgument{
			txidArg: &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: t.resolve,
	}
}

func (t mempool) resolve(p graphql.ResolveParams) (interface{}, error) {
	txid, ok := p.Args[txidArg].(string)
	if ok {
		payload := bytes.Buffer{}

		if txid != "" {
			txidBytes, err := hex.DecodeString(txid)
			if err != nil {
				return nil, errors.New("invalid txid")
			}

			_, _ = payload.Write(txidBytes)
		}

		timeoutGetMempoolTXs := time.Duration(config.Get().Timeout.TimeoutGetMempoolTXs) * time.Second

		resp, err := t.rpcBus.Call(topics.GetMempoolTxs, rpcbus.NewRequest(payload), timeoutGetMempoolTXs)
		if err != nil {
			return "", err
		}

		r := resp.([]txs.ContractCall)
		txs := make([]queryTx, 0)

		for i := 0; i < len(r); i++ {
			d, err := newQueryTx(r[i], nil, 0)
			if err == nil {
				txs = append(txs, d)
			}
		}

		return txs, nil
	}

	return nil, nil
}

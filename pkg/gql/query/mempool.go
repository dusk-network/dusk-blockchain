package query

import (
	"bytes"
	"encoding/hex"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	txs "github.com/dusk-network/dusk-wallet/v2/transactions"
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
			payload.Write(txidBytes)
		}

		resp, err := t.rpcBus.Call(topics.GetMempoolTxs, rpcbus.NewRequest(payload), 5*time.Second)
		if err != nil {
			return "", err
		}
		r := resp.([]txs.Transaction)

		txs := make([]queryTx, 0)
		for i := 0; i < len(r); i++ {
			d, err := newQueryTx(r[i], nil)
			if err == nil {
				txs = append(txs, d)
			}
		}

		return txs, nil
	}

	return nil, nil
}

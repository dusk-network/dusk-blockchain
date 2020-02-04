package query

import (
	"bytes"
	"encoding/hex"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
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
			payload.Write(txidBytes)
		}

		r, err := t.rpcBus.Call(rpcbus.GetMempoolTxs, rpcbus.NewRequest(payload), 5*time.Second)
		if err != nil {
			return "", err
		}

		lTxs, err := encoding.ReadVarInt(&r)
		if err != nil {
			return "", err
		}

		txs := make([]queryTx, 0)
		for i := uint64(0); i < lTxs; i++ {
			tx, err := message.UnmarshalTx(&r)
			if err != nil {
				return "", err
			}

			d, err := newQueryTx(tx, nil)
			if err == nil {
				txs = append(txs, d)
			}
		}

		return txs, nil
	}

	return nil, nil
}

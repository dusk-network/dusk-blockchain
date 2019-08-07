package query

import (
	"bytes"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/graphql-go/graphql"

	rawtxs "github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
)

type mempool struct {
	rpcBus *wire.RPCBus
}

func (t mempool) getQuery() *graphql.Field {
	return &graphql.Field{
		// Slice of Block type which can be found in types.go
		Type: graphql.NewList(Transaction),
		Args: graphql.FieldConfigArgument{
			"txid": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: t.resolve,
	}
}

func (t mempool) resolve(p graphql.ResolveParams) (interface{}, error) {

	txid, ok := p.Args["txid"].(interface{})
	if ok {

		payload := bytes.Buffer{}
		if txid != "" {
			// TODO: Handle case where txid is passed
		}

		r, err := t.rpcBus.Call(wire.GetMempoolTxs, wire.NewRequest(payload, 5))
		if err != nil {
			return "", err
		}

		lTxs, err := encoding.ReadVarInt(&r)
		if err != nil {
			return "", err
		}

		fetched, err := rawtxs.FromReader(&r, lTxs)
		if err != nil {
			return "", err
		}

		txs := make([]rawtxs.Standard, 0)
		for _, tx := range fetched {
			sTx := tx.StandardTX()
			sTx.TxID, err = tx.CalculateHash()
			if err == nil {
				txs = append(txs, sTx)
			}
		}

		return txs, nil
	}

	return nil, nil
}

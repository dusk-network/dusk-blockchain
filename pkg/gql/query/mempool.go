package query

import (
	"bytes"
	"encoding/hex"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/graphql-go/graphql"

	rawtxs "github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
)

type mempool struct {
	rpcBus *rpcbus.RPCBus
}

func (t mempool) getQuery() *graphql.Field {
	return &graphql.Field{
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

	txid, ok := p.Args["txid"].(string)
	if ok {

		payload := bytes.Buffer{}
		if txid != "" {
			txidBytes, err := hex.DecodeString(txid)
			if err != nil {
				return nil, errors.New("invalid txid")
			}
			payload.Write(txidBytes)
		}

		r, err := t.rpcBus.Call(rpcbus.GetMempoolTxs, rpcbus.NewRequest(payload, 5))
		if err != nil {
			return "", err
		}

		lTxs, err := encoding.ReadVarInt(&r)
		if err != nil {
			return "", err
		}

		var fetched []rawtxs.Transaction
		for i := uint64(0); i < lTxs; i++ {
			tx, err := rawtxs.Unmarshal(&r)
			if err != nil {
				return "", err
			}

			fetched = append(fetched, tx)
		}

		txs := make([]*rawtxs.Standard, 0)
		for _, tx := range fetched {
			sTx := tx.StandardTx()
			sTx.TxID, err = tx.CalculateHash()
			if err == nil {
				txs = append(txs, sTx)
			}
		}

		return txs, nil
	}

	return nil, nil
}

package query

import (
	"encoding/base64"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/graphql-go/graphql"

	core "github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
)

type transactions struct {
	db database.DB
}

func (t transactions) getQuery() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(Transaction),
		Args: graphql.FieldConfigArgument{
			"txid": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
			"txids": &graphql.ArgumentConfig{
				Type: graphql.NewList(graphql.String),
			},
		},
		Resolve: t.resolve,
	}
}

func (t transactions) resolve(p graphql.ResolveParams) (interface{}, error) {

	txid, ok := p.Args["txid"].(interface{})
	if ok {
		ids := make([]interface{}, 0)
		ids = append(ids, txid)
		return t.fetchTxsByHash(ids)
	}

	ids, ok := p.Args["txids"].([]interface{})
	if ok {
		return t.fetchTxsByHash(ids)
	}

	return nil, nil
}

type output struct {
	BlockHash []byte
	TxID      []byte
	TxType    core.TxType
}

func (t transactions) fetchTxsByHash(txids []interface{}) ([]output, error) {

	txs := make([]output, 0)
	err := t.db.View(func(t database.Transaction) error {

		for _, v := range txids {
			encVal, ok := v.(string)
			if !ok {
				continue
			}
			decVal, err := base64.StdEncoding.DecodeString(encVal)
			if err != nil {
				return err
			}

			tx, _, hash, err := t.FetchBlockTxByHash(decVal)
			if err != nil {
				return err
			}

			txId, _ := tx.CalculateHash()
			txs = append(txs, output{
				TxID:      txId,
				TxType:    tx.StandardTX().TxType,
				BlockHash: hash,
			})
		}

		return nil
	})

	return txs, err
}

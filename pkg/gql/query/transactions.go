package query

import (
	"encoding/hex"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/graphql-go/graphql"
	"github.com/pkg/errors"

	core "github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
)


type transactions struct {
}

type output struct {
	BlockHash []byte
	TxID      []byte
	TxType    core.TxType
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

	// Retrieve DB conn from context
	db, ok := p.Context.Value("database").(database.DB)
	if !ok {
		return nil, errors.New("context does not store database conn")
	}

	txid, ok := p.Args["txid"].(interface{})
	if ok {
		ids := make([]interface{}, 0)
		ids = append(ids, txid)
		return t.fetchTxsByHash(db, ids)
	}

	ids, ok := p.Args["txids"].([]interface{})
	if ok {
		return t.fetchTxsByHash(db, ids)
	}

	return nil, nil
}

func (t transactions) fetchTxsByHash(db database.DB, txids []interface{}) ([]output, error) {

	txs := make([]output, 0)
	err := db.View(func(t database.Transaction) error {

		for _, v := range txids {
			encVal, ok := v.(string)
			if !ok {
				continue
			}
			decVal, err := hex.DecodeString(encVal)
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

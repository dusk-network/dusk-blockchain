package query

import (
	"encoding/hex"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/graphql-go/graphql"
	"github.com/pkg/errors"

	core "github.com/dusk-network/dusk-wallet/transactions"
	log "github.com/sirupsen/logrus"
)

const (
	txsFetchLimit = 10000
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
			"last": &graphql.ArgumentConfig{
				Type: graphql.Int,
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

	count, ok := p.Args["last"].(int)
	if ok {
		if count <= 0 {
			return nil, errors.New("invalid count")
		}

		return t.fetchLastTxs(db, count)
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
				TxType:    tx.StandardTx().TxType,
				BlockHash: hash,
			})
		}

		return nil
	})

	return txs, err
}

// Fetch #count# number of txs from lastly accepted blocks
func (b transactions) fetchLastTxs(db database.DB, count int) ([]output, error) {

	txs := make([]output, 0)

	if count <= 0 {
		return txs, nil
	}

	if count >= txsFetchLimit {
		msg := fmt.Sprintf("requested txs count exceeds the limit of %d", txsFetchLimit)
		log.Warn(msg)

		return txs, errors.New(msg)
	}

	err := db.View(func(t database.Transaction) error {

		var tip uint64
		tip, err := t.FetchCurrentHeight()
		if err != nil {
			return err
		}

		height := tip

		for {

			hash, err := t.FetchBlockHashByHeight(uint64(height))
			if err != nil {
				return err
			}

			blockTxs, err := t.FetchBlockTxs(hash)
			if err != nil {
				return err
			}

			for _, tx := range blockTxs {

				txId, _ := tx.CalculateHash()
				txs = append(txs, output{
					TxID:      txId,
					TxType:    tx.StandardTx().TxType,
					BlockHash: hash,
				})

				if len(txs) >= count {
					return nil
				}
			}

			if height == 0 {
				break
			}

			height--
		}

		return nil
	})

	return txs, err
}

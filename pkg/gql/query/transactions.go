// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package query

import (
	"bytes"
	"encoding/hex"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/graphql-go/graphql"
	"github.com/pkg/errors"

	core "github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	log "github.com/sirupsen/logrus"
)

const (
	txsFetchLimit = 10000

	txidArg   = "txid"
	txidsArg  = "txids"
	txlastArg = "last"
)

type (
	queryOutput struct {
		PubKey []byte
	}

	queryInput struct {
		KeyImage []byte
	}

	// queryTx is a data-wrapper for all core.transaction relevant fields that
	// can be fetched via grapqhql.
	queryTx struct {
		TxID    []byte
		TxType  core.TxType
		Outputs []queryOutput `json:"output"`
		Inputs  []queryInput  `json:"input"`

		// Non-StandardTx data fields.
		BlockHash []byte
		Size      int
	}
)

type transactions struct{}

// newQueryTx constructs query tx data from core tx and block hash.
//nolint
func newQueryTx(tx core.ContractCall, blockHash []byte) (queryTx, error) {
	txID, err := tx.CalculateHash()
	if err != nil {
		return queryTx{}, err
	}

	qd := queryTx{}
	qd.TxID = txID
	qd.TxType = tx.Type()

	qd.Outputs = make([]queryOutput, 0)
	for _, output := range tx.StandardTx().Notes {

		if IsNil(output) {
			continue
		}

		qd.Outputs = append(qd.Outputs, queryOutput{output.PkR})
	}

	qd.Inputs = make([]queryInput, 0)
	for _, input := range tx.StandardTx().Nullifiers {
		qd.Inputs = append(qd.Inputs, queryInput{input})
	}

	qd.BlockHash = blockHash

	// Populate marshaling size
	buf := new(bytes.Buffer)
	if err := core.Marshal(buf, tx); err != nil {
		return queryTx{}, err
	}

	qd.Size = buf.Len()

	return qd, nil
}

// IsNil will check for nil in a output.
func IsNil(output *core.Note) bool {
	return output.PkR == nil
}

func (t transactions) getQuery() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(Transaction),
		Args: graphql.FieldConfigArgument{
			txidArg: &graphql.ArgumentConfig{
				Type: graphql.String,
			},
			txidsArg: &graphql.ArgumentConfig{
				Type: graphql.NewList(graphql.String),
			},
			txlastArg: &graphql.ArgumentConfig{
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

	txid, ok := p.Args[txidArg].(interface{})
	if ok {
		ids := make([]interface{}, 0)
		ids = append(ids, txid)
		return t.fetchTxsByHash(db, ids)
	}

	ids, ok := p.Args[txidsArg].([]interface{})
	if ok {
		return t.fetchTxsByHash(db, ids)
	}

	count, ok := p.Args[txlastArg].(int)
	if ok {
		if count <= 0 {
			return nil, errors.New("invalid count")
		}

		return t.fetchLastTxs(db, count)
	}

	return nil, nil
}

func (t transactions) fetchTxsByHash(db database.DB, txids []interface{}) ([]queryTx, error) {
	txs := make([]queryTx, 0)
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

			d, err := newQueryTx(tx, hash)
			if err == nil {
				txs = append(txs, d)
			}

		}

		return nil
	})

	return txs, err
}

// Fetch `count` number of txs from lastly accepted blocks.
func (t transactions) fetchLastTxs(db database.DB, count int) ([]queryTx, error) {
	txs := make([]queryTx, 0)

	if count <= 0 {
		return txs, nil
	}

	if count >= txsFetchLimit {
		msg := "requested txs count exceeds the limit"
		log.WithField("txsFetchLimit", txsFetchLimit).
			Warn(msg)
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

			hash, err := t.FetchBlockHashByHeight(height)
			if err != nil {
				return err
			}

			blockTxs, err := t.FetchBlockTxs(hash)
			if err != nil {
				return err
			}

			for _, tx := range blockTxs {

				d, err := newQueryTx(tx, hash)
				if err == nil {
					txs = append(txs, d)
				}

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

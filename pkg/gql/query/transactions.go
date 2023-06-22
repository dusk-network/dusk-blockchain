// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package query

import (
	"bytes"
	"encoding/hex"
	"encoding/json"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/graphql-go/graphql"
	"github.com/pkg/errors"

	core "github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	log "github.com/sirupsen/logrus"
)

const (
	txsFetchLimit       = 10000
	txsBlocksFetchLimit = 10000

	txidArg       = "txid"
	txidsArg      = "txids"
	txlastArg     = "last"
	txblocksArg   = "blocks"
	txblocksRange = "blocksrange"
)

type (
	contractInfo struct {
		Contract []byte
		Method   string
	}

	// queryTx is a data-wrapper for all core.transaction relevant fields that
	// can be fetched via grapqhql.
	queryTx struct {
		TxID     []byte
		TxType   core.TxType
		GasLimit uint64
		GasPrice uint64
		GasSpent uint64

		// Non-StandardTx data fields.
		BlockHash      []byte
		BlockTimestamp int64  `json:"blocktimestamp"` // Block timestamp
		BlockHeight    uint64 `json:"blockheight"`    // Block height
		Size           int
		JSON           string
		TxError        string
		ContractInfo   *contractInfo `json:"contractinfo"`
		Raw            []byte
	}
)

type transactions struct{}

// newQueryTx constructs query tx data from core tx and block hash.
//nolint
func newQueryTx(tx core.ContractCall, blockHash []byte, timestamp int64, blockheight uint64) (queryTx, error) {
	txID, err := tx.CalculateHash()
	if err != nil {
		return queryTx{}, err
	}

	qd := queryTx{}
	qd.TxID = txID
	qd.TxType = tx.Type()

	decoded, err := tx.Decode()
	if err != nil {
		return queryTx{}, err
	}

	qd.GasLimit = decoded.Fee.GasLimit
	qd.GasPrice = decoded.Fee.GasPrice

	qd.GasSpent = tx.GasSpent()
	qd.Raw = tx.StandardTx().Data

	qd.BlockHash = blockHash
	qd.BlockTimestamp = timestamp
	qd.BlockHeight = blockheight

	// Consider Transaction payload length as transaction size
	qd.Size = len(tx.StandardTx().Data)
	b, _ := json.Marshal(decoded)
	qd.JSON = string(b)
	if tx.TxError() != nil {
		b, marshaler := bytes.Buffer{}, jsonpb.Marshaler{}
		if err := marshaler.Marshal(&b, tx.TxError()); err == nil {
			qd.TxError = b.String()
		}
	}

	qd.ContractInfo = decodeContractInfo(decoded)

	return qd, nil
}

func decodeContractInfo(decoded *core.TransactionPayloadDecoded) *contractInfo {
	var info contractInfo = contractInfo{}
	if decoded.Call != nil {
		info.Contract = decoded.Call.ContractID
		info.Method = string(decoded.Call.FnName)
	}
	return &info
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
			txblocksArg: &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
			txblocksRange: &graphql.ArgumentConfig{
				Type: graphql.NewList(graphql.Int),
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

	heightRange, found := p.Args[txblocksRange].([]interface{})
	if found && len(heightRange) == 2 {
		from, isint := heightRange[0].(int)
		if !isint {
			return nil, errors.New("range `from` value not int64")
		}

		to, isint := heightRange[1].(int)
		if !isint {
			return nil, errors.New("range `to` value not int64")
		}

		return t.fetchTxsByBlocksHeights(db, int64(from), int64(to))
	}

	count, ok := p.Args[txlastArg].(int)
	if ok {
		if count <= 0 {
			return nil, errors.New("invalid ``" + txlastArg + "`` argument")
		}

		maxBlocks, ok := p.Args[txblocksArg].(int)
		if !ok {
			maxBlocks = txsBlocksFetchLimit
		}
		return t.fetchLastTxs(db, count, maxBlocks)
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

			header, err := t.FetchBlockHeader(hash)
			if err != nil {
				return err
			}

			d, err := newQueryTx(tx, header.Hash, header.Timestamp, header.Height)
			if err == nil {
				txs = append(txs, d)
			}
		}

		return nil
	})

	return txs, err
}

// Fetch `count` number of txs from lastly `maxBlocks` accepted blocks.
func (t transactions) fetchLastTxs(db database.DB, count int, maxBlocks int) ([]queryTx, error) {
	txs := make([]queryTx, 0)

	if count <= 0 || maxBlocks <= 0 {
		return txs, nil
	}

	if count > txsFetchLimit {
		msg := "requested txs count exceeds the limit"
		log.WithField("txsFetchLimit", txsFetchLimit).
			Warn(msg)
		return txs, errors.New(msg)
	}

	if maxBlocks > txsBlocksFetchLimit {
		msg := "requested txs MaxBlock count exceeds the limit"
		log.WithField("txsBlocksFetchLimit", txsBlocksFetchLimit).
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

			header, err := t.FetchBlockHeader(hash)
			if err != nil {
				return err
			}

			for _, tx := range blockTxs {
				d, err := newQueryTx(tx, header.Hash, header.Timestamp, header.Height)
				if err == nil {
					txs = append(txs, d)
				}

				if len(txs) >= count {
					return nil
				}
			}

			maxBlocks--
			if maxBlocks <= 0 {
				break
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

// Fetch all txs within a range of block heights.
func (t transactions) fetchTxsByBlocksHeights(db database.DB, from, to int64) ([]queryTx, error) {
	txs := make([]queryTx, 0)

	if from > to {
		msg := "invalid range"
		log.WithField("from", from).
			WithField("to", to).
			Warn(msg)
		return txs, errors.New(msg)
	}

	if (to - from) > txsBlocksFetchLimit {
		msg := "requested txs blocks count exceeds the limit"
		log.WithField("txsBlocksFetchLimit", txsBlocksFetchLimit).
			Warn(msg)
		return txs, errors.New(msg)
	}

	err := db.View(func(t database.Transaction) error {
		tip, err := t.FetchCurrentHeight()
		if err != nil {
			return err
		}
		for height := from; height <= to && uint64(height) <= tip; height++ {
			hash, err := t.FetchBlockHashByHeight(uint64(height))
			if err != nil {
				return err
			}

			blockTxs, err := t.FetchBlockTxs(hash)
			if err != nil {
				return err
			}

			header, err := t.FetchBlockHeader(hash)
			if err != nil {
				return err
			}

			for _, tx := range blockTxs {
				d, err := newQueryTx(tx, header.Hash, header.Timestamp, header.Height)
				if err == nil {
					txs = append(txs, d)
				}
			}

		}

		return nil
	})

	return txs, err
}

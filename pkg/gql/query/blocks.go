// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package query

import (
	"encoding/hex"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/graphql-go/graphql"
)

const (
	blockHashArg   = "hash"
	blockHashesArg = "hashes"
	blockHeightArg = "height"
	blockRangeArg  = "range"
	blockLastArg   = "last"
	blockSinceArg  = "since"
)

// File purpose is to define all arguments and resolvers relevant to "blocks" query only.

type blocks struct{}

func (b blocks) getQuery() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewList(Block),
		Args: graphql.FieldConfigArgument{
			blockHashArg: &graphql.ArgumentConfig{
				Type: graphql.String,
			},
			blockHashesArg: &graphql.ArgumentConfig{
				Type: graphql.NewList(graphql.String),
			},
			blockHeightArg: &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
			blockRangeArg: &graphql.ArgumentConfig{
				Type: graphql.NewList(graphql.Int),
			},
			blockLastArg: &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
			blockSinceArg: &graphql.ArgumentConfig{
				Type: graphql.DateTime,
			},
		},
		Resolve: b.resolve,
	}
}

func (b blocks) resolve(p graphql.ResolveParams) (interface{}, error) {
	// Retrieve DB conn from context
	db, ok := p.Context.Value("database").(database.DB)
	if !ok {
		return nil, errors.New("context does not store database conn")
	}

	// resolve argument hash (single block)
	hash, ok := p.Args[blockHashArg].(interface{})
	if ok {
		hashes := make([]interface{}, 0)
		hashes = append(hashes, hash)
		return b.fetchBlocksByHashes(db, hashes)
	}

	// resolve argument hashes (multiple blocks)
	hashes, ok := p.Args[blockHashesArg].([]interface{})
	if ok {
		return b.fetchBlocksByHashes(db, hashes)
	}

	// resolve argument height (single block)
	// Chain height type is uint64 whereas `resolve` can handle height values up to MaxUInt
	height, ok := p.Args[blockHeightArg].(int)
	if ok {
		return b.fetchBlocksByHeights(db, int64(height), int64(height))
	}

	// resolve argument range (range of blocks)
	heightRange, found := p.Args[blockRangeArg].([]interface{})
	if found && len(heightRange) == 2 {
		from, ok := heightRange[0].(int)
		if !ok {
			return nil, errors.New("range `from` value not int64")
		}

		to, ok := heightRange[1].(int)
		if !ok {
			return nil, errors.New("range `to` value not int64")
		}

		return b.fetchBlocksByHeights(db, int64(from), int64(to))
	}

	offset, offsetOK := p.Args[blockLastArg].(int)
	if offsetOK {
		if offset <= 0 {
			return nil, errors.New("invalid offset")
		}
		return b.fetchBlocksByHeights(db, int64(offset)*-1, -1)
	}

	date, dateOK := p.Args[blockSinceArg].(time.Time)
	if dateOK {
		return b.fetchBlocksByDate(db, date)
	}

	return nil, nil
}

func resolveTxs(p graphql.ResolveParams) (interface{}, error) {
	txs := make([]queryTx, 0)

	b, ok := p.Source.(*block.Block)
	if ok {
		// Retrieve DB conn from context
		db, ok := p.Context.Value("database").(database.DB)
		if !ok {
			return nil, errors.New("context does not store database conn")
		}

		err := db.View(func(t database.Transaction) error {
			fetched, err := t.FetchBlockTxs(b.Header.Hash)
			if err != nil {
				return err
			}

			for _, tx := range fetched {
				d, err := newQueryTx(tx, b.Header.Hash)
				if err == nil {
					txs = append(txs, d)
				}
			}

			return nil
		})

		return txs, err
	}

	return nil, errors.New("invalid source block")
}

// Fetch block headers by a list of hashes.
func (b blocks) fetchBlocksByHashes(db database.DB, hashes []interface{}) ([]*block.Block, error) {
	blocks := make([]*block.Block, 0)
	err := db.View(func(t database.Transaction) error {
		for _, v := range hashes {
			encodedHash, ok := v.(string)
			if !ok {
				continue
			}

			hash, err := hex.DecodeString(encodedHash)
			if err != nil {
				continue
			}

			header, err := t.FetchBlockHeader(hash)
			if err != nil {
				continue
			}

			// Reconstructing block with header only
			b := &block.Block{
				Header: header,
				Txs:    nil,
			}

			blocks = append(blocks, b)
		}
		return nil
	})

	return blocks, err
}

// Fetch block headers by a range of heights.
func (b blocks) fetchBlocksByHeights(db database.DB, from, to int64) ([]*block.Block, error) {
	blocks := make([]*block.Block, 0)
	err := db.View(func(t database.Transaction) error {
		var tip uint64
		if from < 0 || to < 0 {
			var err error
			tip, err = t.FetchCurrentHeight()
			if err != nil {
				return err
			}
		}

		if from == -1 {
			from = int64(tip)
		}

		// For cases where last N blocks are required
		if from < 0 {
			from = int64(tip) + from + 1
			if from < 0 {
				from = 0
			}
		}

		if to == -1 {
			to = int64(tip)
		}

		for height := from; height <= to; height++ {
			hash, err := t.FetchBlockHashByHeight(uint64(height))
			if err != nil {
				return err
			}

			header, err := t.FetchBlockHeader(hash)
			if err != nil {
				continue
			}

			// Reconstructing block with header only
			b := &block.Block{
				Header: header,
				Txs:    nil,
			}

			blocks = append(blocks, b)
		}
		return nil
	})

	return blocks, err
}

// Fetch block headers by a range of heights.
func (b blocks) fetchBlocksByDate(db database.DB, sinceDate time.Time) ([]*block.Block, error) {
	blocks := make([]*block.Block, 0)

	err := db.View(func(t database.Transaction) error {
		const offset = 24 * 3600
		height, err := t.FetchBlockHeightSince(sinceDate.Unix(), offset)
		if err != nil {
			return err
		}

		hash, err := t.FetchBlockHashByHeight(height)
		if err != nil {
			return err
		}

		header, err := t.FetchBlockHeader(hash)
		if err != nil {
			return err
		}

		// Reconstructing block with header only
		b := &block.Block{
			Header: header,
			Txs:    nil,
		}

		blocks = append(blocks, b)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return blocks, err
}

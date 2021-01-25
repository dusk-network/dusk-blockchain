// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package notifications

import (
	"encoding/hex"
	"encoding/json"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
)

// BlockMsg represents the data need by Explorer UI on each new block accepted.
type BlockMsg struct {
	Height    uint64
	Hash      string
	Timestamp int64
	Txs       []string

	// BlocksGeneratedCount is number of blocks generated last 24 hours.
	BlocksGeneratedCount uint
}

// MarshalBlockMsg builds the JSON from a subset of block fields.
func MarshalBlockMsg(blk block.Block) (string, error) {
	hash, err := blk.CalculateHash()
	if err != nil {
		return "", err
	}

	var p BlockMsg
	p.Height = blk.Header.Height
	p.Timestamp = blk.Header.Timestamp
	p.Hash = hex.EncodeToString(hash)
	p.Txs = make([]string, 0)

	// Get a limited set of block txs hashes
	for _, tx := range blk.Txs {
		if len(p.Txs) >= maxTxsPerMsg {
			break
		}

		txid, e := tx.CalculateHash()
		if e != nil {
			return "", e
		}

		p.Txs = append(p.Txs, hex.EncodeToString(txid))
	}

	msg, marshalErr := json.Marshal(p)
	if marshalErr != nil {
		return "", marshalErr
	}

	return string(msg), nil
}

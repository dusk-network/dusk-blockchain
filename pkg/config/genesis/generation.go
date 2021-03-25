// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package genesis

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

// Generate a genesis block. The constitution of the block depends on the passed
// config.
func Generate(c Config) *block.Block {
	h := &block.Header{
		Version:       0,
		Timestamp:     c.timestamp,
		Height:        0,
		PrevBlockHash: make([]byte, 32),
		TxRoot:        nil,
		Seed:          c.seed,
		Certificate:   block.EmptyCertificate(),
	}

	txs := make([]transactions.ContractCall, 0)

	for i := uint(0); i < c.initialCommitteeSize; i++ {
		buf := new(bytes.Buffer)
		if err := encoding.WriteUint64LE(buf, 250000); err != nil {
			panic(err)
		}

		if err := encoding.WriteVarBytes(buf, c.committeeMembers[i]); err != nil {
			panic(err)
		}

		stake := transactions.NewTransaction()
		stake.Payload.CallData = buf.Bytes()
		amount := c.stakeValue * wallet.DUSK
		amountBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(amountBytes[0:8], amount)

		stake.Payload.Notes = append(stake.Payload.Notes, &transactions.Note{
			Randomness:    make([]byte, 32),
			PkR:           c.initialParticipants[i].AG,
			Commitment:    amountBytes,
			Nonce:         make([]byte, 32),
			EncryptedData: make([]byte, 96),
		})

		stake.TxType = transactions.Stake
		txs = append(txs, stake)
	}

	for i := uint(0); i < c.initialBlockGenerators; i++ {
		buf := new(bytes.Buffer)
		if err := encoding.WriteUint64LE(buf, 250000); err != nil {
			panic(err)
		}

		m := make([]byte, 32)
		if err := encoding.Write256(buf, m); err != nil {
			panic(err)
		}

		bid := transactions.NewTransaction()
		bid.Payload.CallData = buf.Bytes()
		amount := c.bidValue * wallet.DUSK
		amountBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(amountBytes[0:8], amount)

		bid.Payload.Notes = append(bid.Payload.Notes, &transactions.Note{
			Randomness:    make([]byte, 32),
			PkR:           c.initialParticipants[i].AG,
			Commitment:    amountBytes,
			Nonce:         make([]byte, 32),
			EncryptedData: make([]byte, 96),
		})
		bid.TxType = transactions.Bid
		txs = append(txs, bid)
	}

	for _, pk := range c.initialParticipants {
		// Add 200 coinbase outputs
		for i := uint(0); i < c.coinbaseAmount; i++ {
			buf := new(bytes.Buffer)
			if err := encoding.WriteUint64LE(buf, c.coinbaseValue*wallet.DUSK); err != nil {
				panic(err)
			}

			amount := c.coinbaseValue * wallet.DUSK
			amountBytes := make([]byte, 32)
			binary.LittleEndian.PutUint64(amountBytes[0:8], amount)

			coinbase := transactions.NewTransaction()
			coinbase.Payload.CallData = buf.Bytes()
			coinbase.Payload.Notes = append(coinbase.Payload.Notes, &transactions.Note{
				Randomness:    make([]byte, 32),
				PkR:           pk.AG,
				Commitment:    amountBytes,
				Nonce:         make([]byte, 32),
				EncryptedData: make([]byte, 96),
			})
			coinbase.TxType = transactions.Distribute
			txs = append(txs, coinbase)
		}
	}

	b := &block.Block{
		Header: h,
		Txs:    txs,
	}

	// Set root and hash, since they have changed because of the adding of txs.
	root, err := b.CalculateRoot()
	if err != nil {
		panic(err)
	}

	b.Header.TxRoot = root

	hash, err := b.CalculateHash()
	if err != nil {
		panic(err)
	}

	b.Header.Hash = hash
	return b
}

// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package block

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// Block defines a block on the Dusk blockchain.
type Block struct {
	Header *Header                     `json:"header"`
	Txs    []transactions.ContractCall `json:"transactions"`
}

// NewBlock will return an empty Block with an empty BlockHeader.
func NewBlock() *Block {
	return &Block{
		Header: NewHeader(),
	}
}

// IsZero will return if Block has a zero-ed header.
func (b Block) IsZero() bool {
	if len(b.Header.StateHash) != 32 {
		return false
	}

	if bytes.Equal(b.Header.StateHash, EmptyHash[:]) {
		return true
	}

	if len(b.Header.Hash) != 32 {
		return false
	}

	if bytes.Equal(b.Header.Hash, EmptyHash[:]) {
		return true
	}

	return false
}

// IsEmpty tells us if a block is empty. We can check this easily, because an
// empty block will be created as `block.Block{}`, meaning that the header
// is always nil. This function basically checks the existence of the block
// header.
func (b Block) IsEmpty() bool {
	return b.Header == nil
}

// Copy returns a deep copy of the Block safe to publish to multiple subscribers.
func (b Block) Copy() payload.Safe {
	cpy := Block{}
	cpy.Header = b.Header.Copy()

	if b.Txs != nil {
		cpy.Txs = make([]transactions.ContractCall, len(b.Txs))
		for i, tx := range b.Txs {
			cpy.Txs[i] = tx.Copy().(transactions.ContractCall)
		}
	}

	return cpy
}

// SetPrevBlock will set all the previous block hash field from a header.
func (b *Block) SetPrevBlock(prevHeader *Header) {
	b.Header.PrevBlockHash = prevHeader.Hash
}

// AddTx will add a transaction to the block.
func (b *Block) AddTx(tx *transactions.Transaction) {
	b.Txs = append(b.Txs, tx)
}

// Clear will empty out all the block's fields.
func (b *Block) Clear() {
	b.Header = &Header{}
	b.Txs = nil
}

// CalculateHash will calculate the block hash.
func (b *Block) CalculateHash() ([]byte, error) {
	return b.Header.CalculateHash()
}

// Equals returns true if two blocks are equal.
func (b *Block) Equals(other *Block) bool {
	if other == nil {
		return false
	}

	if !b.Header.Equals(other.Header) {
		return false
	}

	if len(b.Txs) != len(other.Txs) {
		return false
	}

	for i := 0; i < len(b.Txs); i++ {
		if !transactions.Equal(b.Txs[i], other.Txs[i]) {
			return false
		}
	}

	return true
}

// Tx returns transaction by id if exists.
func (b Block) Tx(txid []byte) (transactions.ContractCall, error) {
	if b.Txs != nil {
		for _, tx := range b.Txs {
			h, err := tx.CalculateHash()
			if err != nil {
				continue
			}

			if bytes.Equal(h, txid) {
				return tx, nil
			}
		}
	}

	return nil, errors.New("not found")
}

// TamperExecutedTransaction tamper gasSpent and txError of a transaction specified by txid.
func (b *Block) TamperExecutedTransaction(txid []byte, gasSpent uint64, txError *rusk.ExecutedTransaction_Error) error {
	if b.Txs == nil {
		return errors.New("not found")
	}

	for i, tx := range b.Txs {
		h, err := tx.CalculateHash()
		if err != nil {
			continue
		}

		if !bytes.Equal(h, txid) {
			continue
		}

		// Specified tx found.
		updated, err := transactions.UpdateTransaction(tx, gasSpent, txError)
		if err != nil {
			return err
		}

		b.Txs[i] = updated

		return nil
	}

	return errors.New("not found")
}

package generation

import (
	"bytes"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	BlockGenerator interface {
		GenerateBlock(uint64, []byte) (*block.Block, error)
	}

	blockGenerator struct {
		rpcBus *wire.RPCBus
	}
)

func newBlockGenerator(rpcBus *wire.RPCBus) *blockGenerator {
	return &blockGenerator{
		rpcBus: rpcBus,
	}
}

func (bg *blockGenerator) GenerateBlock(round uint64, seed []byte) (*block.Block, error) {
	// TODO Missing fields for forging the block
	// - CertHash
	// - prevHash

	// Retrieve latest verified transactions from Mempool
	r, err := bg.rpcBus.Call(wire.GetVerifiedTxs, wire.NewRequest(bytes.Buffer{}, 10))
	if err != nil {
		return nil, err
	}

	lTxs, err := encoding.ReadVarInt(&r)
	if err != nil {
		return nil, err
	}

	txs, err := transactions.FromReader(&r, lTxs)
	if err != nil {
		return nil, err
	}

	h := &block.Header{
		Version:   0,
		Timestamp: time.Now().Unix(),
		Height:    round,
		// TODO: store previous block hash on generation component somewhere
		PrevBlock: make([]byte, 32),
		TxRoot:    nil,

		Seed:     nil,
		CertHash: nil,
	}

	// Generate the candidate block
	b := &block.Block{
		Header: h,
		Txs:    txs,
	}

	// Update TxRoot
	if err := b.SetRoot(); err != nil {
		return nil, err
	}

	// Generate the block hash
	if err := b.SetHash(); err != nil {
		return nil, err
	}

	// Ensure the forged block satisfies all chain rules
	// if err := verifiers.CheckBlock(c.db, c.prevBlock, *b); err != nil {
	// 	return nil, err
	// }

	// Save it into persistent storage
	// err = c.db.Update(func(t database.Transaction) error {
	// 	err := t.StoreCandidateBlock(b)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	return nil
	// })

	// if err != nil {
	// 	return nil, err
	// }

	return b, nil
}

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
	// BlockGenerator defines a method which will create and return a new block,
	// given a height and seed.
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
	// - PrevBlock

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
		// TODO: store/get previous block hash from somewhere
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

	// TODO: maybe verify the block before returning it
	return b, nil
}

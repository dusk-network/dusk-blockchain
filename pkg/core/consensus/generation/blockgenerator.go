package generation

import (
	"bytes"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	blockGenerator struct {
		stakePool map[string]*transactions.Stake
		bidPool   map[string]*transactions.Bid
	}
)

func newBlockGenerator(eventBroker wire.EventBroker) *blockGenerator {
	bg := &blockGenerator{
		stakePool: make(map[string]*transactions.Stake),
		bidPool:   make(map[string]*transactions.Bid),
	}

	eventBroker.SubscribeCallback(string(msg.StakeTopic), bg.CollectStake)
	eventBroker.SubscribeCallback(string(msg.BidTopic), bg.CollectBid)
	return bg
}

func (b *blockGenerator) CollectStake(message *bytes.Buffer) error {
	copyBuf := *message
	txs, err := transactions.FromReader(&copyBuf, 1)
	if err != nil {
		return err
	}

	tx := txs[0].(*transactions.Stake)
	b.stakePool[hex.EncodeToString(tx.PubKeyBLS)] = tx
	return nil
}

func (b *blockGenerator) CollectBid(message *bytes.Buffer) error {
	copyBuf := *message
	txs, err := transactions.FromReader(&copyBuf, 1)
	if err != nil {
		return err
	}

	tx := txs[0].(*transactions.Bid)
	b.bidPool[hex.EncodeToString(tx.M)] = tx
	return nil
}

func (b *blockGenerator) generateBlock(round uint64, seed []byte) *block.Block {
	cert, _ := crypto.RandEntropy(32)
	blk := &block.Block{
		Header: &block.Header{
			Version:   0,
			Timestamp: time.Now().Unix(),
			Height:    round,
			PrevBlock: make([]byte, 32),
			Seed:      seed,
			TxRoot:    make([]byte, 32),
			CertHash:  cert,
		},
		Txs: make([]transactions.Transaction, 0),
	}

	for _, tx := range b.stakePool {
		blk.AddTx(tx)
	}

	for _, tx := range b.bidPool {
		blk.AddTx(tx)
	}

	if len(blk.Txs) > 0 {
		if err := blk.SetRoot(); err != nil {
			panic(err)
		}
	}

	return blk
}

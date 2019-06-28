package main

import (
	"bytes"
	"encoding/hex"
	"time"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// TODO: This source file should be deleted when getting ready for testnet
func mockBlockOne(bid *transactions.Bid, stake *transactions.Stake) *block.Block {
	blk := block.NewBlock()
	blk.Header.Height = 1
	blk.Header.Timestamp = time.Now().Unix()
	coinbase := mockCoinbaseTx()
	blk.AddTx(coinbase)
	blk.AddTx(bid)
	blk.AddTx(stake)

	genesisBlock := getGenesisBlock()
	blk.SetPrevBlock(genesisBlock.Header)

	seed, _ := crypto.RandEntropy(33)
	blk.Header.Seed = seed
	if err := blk.SetRoot(); err != nil {
		panic(err)
	}

	if err := blk.SetHash(); err != nil {
		panic(err)
	}

	return blk
}

func mockCoinbaseTx() *transactions.Coinbase {
	proof, _ := crypto.RandEntropy(2000)
	score, _ := crypto.RandEntropy(32)
	r, _ := crypto.RandEntropy(32)
	coinbase := transactions.NewCoinbase(proof, score, r)

	commitment := make([]byte, 32)
	commitment[0] = 100
	output, err := transactions.NewOutput(commitment, r, proof)
	if err != nil {
		panic(err)
	}

	coinbase.AddReward(output)
	return coinbase
}

func getGenesisBlock() *block.Block {
	genesisBlock := block.NewBlock()
	blob, err := hex.DecodeString(cfg.TestNetGenesisBlob)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	buf.Write(blob)
	if err := genesisBlock.Decode(&buf); err != nil {
		panic(err)
	}

	return genesisBlock
}

func waitForStake(bus *wire.EventBus, myStake *transactions.Stake) uint64 {
	blockChan := make(chan *bytes.Buffer, 100)
	id := bus.Subscribe(string(topics.AcceptedBlock), blockChan)
	for {
		blkBuf := <-blockChan
		blk := block.NewBlock()
		if err := blk.Decode(blkBuf); err != nil {
			panic(err)
		}

		for _, tx := range blk.Txs {
			if tx.Equals(myStake) {
				bus.Unsubscribe(string(topics.AcceptedBlock), id)
				return blk.Header.Height
			}
		}
	}
}

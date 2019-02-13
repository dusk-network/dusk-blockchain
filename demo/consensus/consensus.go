package main

import (
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
)

func setupConsensusConfig() *consensus.Config {
	return &consensus.Config{
		GetLatestHeight:        getLatestHeight,
		GetBlockHeaderByHeight: getBlockHeaderByHeight,
		GetBlock:               getBlock,
		VerifyBlock:            verifyBlock,
	}
}

func getLatestHeight() uint64 {
	return 1
}

func getBlockHeaderByHeight(height uint64) (block.Header, error) {
	prevBlock, err := crypto.RandEntropy(32)
	if err != nil {
		return block.Header{}, err
	}

	seed, err := crypto.RandEntropy(32)
	if err != nil {
		return block.Header{}, err
	}

	txRoot, err := crypto.RandEntropy(32)
	if err != nil {
		return block.Header{}, err
	}

	certHash, err := crypto.RandEntropy(32)
	if err != nil {
		return block.Header{}, err
	}

	hash, err := crypto.RandEntropy(32)
	if err != nil {
		return block.Header{}, err
	}

	return block.Header{
		Version:   1,
		Timestamp: time.Now().Unix(),
		Height:    height,
		PrevBlock: prevBlock,
		Seed:      seed,
		TxRoot:    txRoot,
		CertHash:  certHash,
		Hash:      hash,
	}, nil
}

func getBlock(header []byte) (block.Block, error) {
	return block.Block{}, nil
}

func verifyBlock(blk *block.Block) error {
	return nil
}

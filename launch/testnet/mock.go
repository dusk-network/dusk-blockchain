package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math"
	"math/big"
	"math/rand"
	"time"

	ristretto "github.com/bwesterb/go-ristretto"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/zkproof"
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

func makeStake(keys *user.Keys) *transactions.Stake {
	stake, _ := transactions.NewStake(0, math.MaxUint64, 100, *keys.EdPubKey, keys.BLSPubKey.Marshal())
	keyImage, _ := crypto.RandEntropy(32)
	txID, _ := crypto.RandEntropy(32)
	signature, _ := crypto.RandEntropy(32)
	input, _ := transactions.NewInput(keyImage, txID, 0, signature)
	stake.Inputs = transactions.Inputs{input}

	outputAmount := rand.Int63n(100000)
	commitment := make([]byte, 32)
	binary.BigEndian.PutUint64(commitment[24:32], uint64(outputAmount))
	destKey, _ := crypto.RandEntropy(32)
	rangeProof, _ := crypto.RandEntropy(32)
	output, _ := transactions.NewOutput(commitment, destKey, rangeProof)
	stake.Outputs = transactions.Outputs{output}

	return stake
}

func makeBid() (*transactions.Bid, ristretto.Scalar, ristretto.Scalar) {
	k := ristretto.Scalar{}
	k.Rand()
	outputAmount := rand.Int63n(100000)
	d := big.NewInt(outputAmount)
	dScalar := ristretto.Scalar{}
	dScalar.SetBigInt(d)
	m := zkproof.CalculateM(k)
	bid, _ := transactions.NewBid(0, math.MaxUint64, 100, m.Bytes())
	keyImage, _ := crypto.RandEntropy(32)
	txID, _ := crypto.RandEntropy(32)
	signature, _ := crypto.RandEntropy(32)
	input, _ := transactions.NewInput(keyImage, txID, 0, signature)
	bid.Inputs = transactions.Inputs{input}

	commitment := make([]byte, 32)
	binary.BigEndian.PutUint64(commitment[24:32], uint64(outputAmount))
	destKey, _ := crypto.RandEntropy(32)
	rangeProof, _ := crypto.RandEntropy(32)
	output, _ := transactions.NewOutput(commitment, destKey, rangeProof)
	bid.Outputs = transactions.Outputs{output}

	return bid, dScalar, k
}

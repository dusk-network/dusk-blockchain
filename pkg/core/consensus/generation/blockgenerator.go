package generation

import (
	"bytes"
	"math/big"
	"time"

	"github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// BlockGenerator defines a method which will create and return a new block,
	// given a height and seed.
	BlockGenerator interface {
		GenerateBlock(round uint64, seed, proof, score, prevBlockHash []byte) (*block.Block, error)
	}

	blockGenerator struct {
		// generator Public Keys to sign the rewards tx
		genPubKey *key.PublicKey
		rpcBus    *wire.RPCBus
	}
)

func newBlockGenerator(genPubKey *key.PublicKey, rpcBus *wire.RPCBus) *blockGenerator {
	return &blockGenerator{
		rpcBus:    rpcBus,
		genPubKey: genPubKey,
	}
}

func (bg *blockGenerator) GenerateBlock(round uint64, seed, proof, score, prevBlockHash []byte) (*block.Block, error) {
	txs, err := bg.ConstructBlockTxs(proof, score)
	if err != nil {
		return nil, err
	}

	// Construct header
	h := &block.Header{
		Version:       0,
		Timestamp:     time.Now().Unix(),
		Height:        round,
		PrevBlockHash: prevBlockHash,
		TxRoot:        nil,
		Seed:          seed,
		Certificate:   block.EmptyCertificate(),
	}

	// Construct the candidate block
	candidateBlock := &block.Block{
		Header: h,
		Txs:    txs,
	}

	// Update TxRoot
	if err := candidateBlock.SetRoot(); err != nil {
		return nil, err
	}

	// Generate the block hash
	if err := candidateBlock.SetHash(); err != nil {
		return nil, err
	}

	return candidateBlock, nil
}

func (bg *blockGenerator) ConstructBlockTxs(proof, score []byte) ([]transactions.Transaction, error) {

	txs := make([]transactions.Transaction, 0)

	// Construct and append coinbase Tx to reward the generator
	coinbaseTx, err := bg.constructCoinbaseTx(bg.genPubKey, proof, score)
	if err != nil {
		return nil, err
	}

	txs = append(txs, coinbaseTx)

	// Retrieve and append the verified transactions from Mempool
	if bg.rpcBus != nil {
		r, err := bg.rpcBus.Call(wire.GetMempoolTxs, wire.NewRequest(bytes.Buffer{}, 10))
		// TODO: GetVerifiedTxs should ensure once again that none of the txs have been
		// already accepted in the the chain.
		if err != nil {
			return nil, err
		}

		lTxs, err := encoding.ReadVarInt(&r)
		if err != nil {
			return nil, err
		}

		mempoolTxs, err := transactions.FromReader(&r, lTxs)
		if err != nil {
			return nil, err
		}

		txs = append(txs, mempoolTxs...)
	}

	// TODO Append Provisioners rewards

	return txs, nil
}

// constructCoinbaseTx forges the transactions to reward the block generator
func (c *blockGenerator) constructCoinbaseTx(rewardReceiver *key.PublicKey, proof []byte, score []byte) (*transactions.Coinbase, error) {
	// The rewards for both the Generator and the Provisioners are disclosed.
	// Provisioner reward addresses do not require obfuscation
	// The Generator address rewards do.

	// Construct one-time address derived from block generator public key
	// the big random number to be used in calculating P and R
	var r ristretto.Scalar
	r.Rand()

	// The transaction is broadcast along with R=rG
	var R ristretto.Point
	R.ScalarMultBase(&r)

	// Store the reward in the coinbase tx
	tx := transactions.NewCoinbase(proof, score, R.Bytes())

	// To sign reward output, we calculate P = H(rA)G + B where
	//
	// A is the BlockGenerator's PubView key (BlockGenerator's public key 1)
	// B is the BlockGenerator's SpendView key (BlockGenerator's public key 2)
	// r is a random big number
	// G is the curve generator point
	// H is hashing function
	const coinbaseIndex = 0
	P := rewardReceiver.StealthAddress(r, coinbaseIndex).P

	// Disclose  reward
	var reward ristretto.Scalar
	reward.SetBigInt(big.NewInt(int64(config.GeneratorReward)))

	output := &transactions.Output{
		// EncryptedAmount field in coinbase tx represents the reward
		EncryptedAmount: reward.Bytes(),
		EncryptedMask:   make([]byte, 1),
		Commitment:      make([]byte, 32),
		DestKey:         P.Bytes(),
	}

	tx.AddReward(output)

	// TODO: Optional here could be to verify if the reward is spendable by the generator wallet.
	// This could be achieved with a request to dusk-wallet

	return tx, nil
}

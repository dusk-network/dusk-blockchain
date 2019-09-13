package generation

import (
	"bytes"
	"math/big"
	"time"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
	"github.com/dusk-network/dusk-wallet/key"
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
		rpcBus    *rpcbus.RPCBus
	}
)

func newBlockGenerator(genPubKey *key.PublicKey, rpcBus *rpcbus.RPCBus) *blockGenerator {
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
		r, err := bg.rpcBus.Call(rpcbus.GetMempoolTxs, rpcbus.NewRequest(bytes.Buffer{}, 10))
		// TODO: GetVerifiedTxs should ensure once again that none of the txs have been
		// already accepted in the the chain.
		if err != nil {
			return nil, err
		}

		lTxs, err := encoding.ReadVarInt(&r)
		if err != nil {
			return nil, err
		}

		for i := uint64(0); i < lTxs; i++ {
			tx, err := transactions.Unmarshal(&r)
			if err != nil {
				return nil, err
			}

			txs = append(txs, tx)
		}
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

	// Create transaction
	tx := transactions.NewCoinbase(proof, score, 2)

	// Set r to our generated value
	tx.SetTxPubKey(r)

	// Disclose  reward
	var reward ristretto.Scalar
	reward.SetBigInt(big.NewInt(int64(config.GeneratorReward)))

	// Store the reward in the coinbase tx
	tx.AddReward(*rewardReceiver, reward)

	// TODO: Optional here could be to verify if the reward is spendable by the generator wallet.
	// This could be achieved with a request to dusk-wallet

	return tx, nil
}

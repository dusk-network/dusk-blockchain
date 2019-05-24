package generation

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/verifiers"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
)

type (
	BlockGenerator interface {
		GenerateBlock(round uint64, seed []byte, proof []byte, score []byte) (*block.Block, error)
		UpdatePrevBlock(b block.Block)
	}

	blockGenerator struct {
		// generator Public Keys to sign the rewards tx
		genPubKey *key.PublicKey

		db     database.DB
		rpcBus *wire.RPCBus

		prevBlock block.Block
	}
)

func newBlockGenerator(genPubKey *key.PublicKey, rpcBus *wire.RPCBus) *blockGenerator {

	drvr, err := database.From(cfg.Get().Database.Driver)
	if err != nil {
		panic(err)
	}

	// TODO: Consider closing this db connection on-time
	db, err := drvr.Open(cfg.Get().Database.Dir, protocol.MagicFromConfig(), false)
	if err != nil {
		panic(err)
	}

	return &blockGenerator{
		rpcBus:    rpcBus,
		db:        db,
		genPubKey: genPubKey,
		prevBlock: *block.NewBlock(),
	}
}

func (bg *blockGenerator) UpdatePrevBlock(b block.Block) {
	bg.prevBlock = b
}

func (bg *blockGenerator) GenerateBlock(round uint64, seed []byte, proof []byte, score []byte) (*block.Block, error) {

	if round <= bg.prevBlock.Header.Height {
		return nil, fmt.Errorf("target round (%d) must be higher than previous block round %d", round, bg.prevBlock.Header.Height)
	}

	// TODO Missing fields for forging the block
	// - CertHash

	certHash, _ := crypto.RandEntropy(32)

	txs, err := bg.ConstructBlockTxs(proof, score)
	if err != nil {
		return nil, err
	}

	// Construct header
	h := &block.Header{
		Version:   0,
		Timestamp: time.Now().Unix(),
		Height:    round,
		PrevBlock: bg.prevBlock.Header.Hash,
		TxRoot:    nil,

		Seed:     seed,
		CertHash: certHash,
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

	// Ensure the forged block satisfies all chain rules
	if err := verifiers.CheckBlock(bg.db, bg.prevBlock, *candidateBlock); err != nil {
		return nil, err
	}

	// Save it into persistent storage
	err = bg.db.Update(func(t database.Transaction) error {
		err := t.StoreCandidateBlock(candidateBlock)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
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
	r, err := bg.rpcBus.Call(wire.GetVerifiedTxs, wire.NewRequest(bytes.Buffer{}, 10))
	// TODO: GetVerifiedTxs should ensure once again that none of the txs have been
	// already accepted in the the chain.
	if err != nil {
		return nil, err
	}

	lTxs, err := encoding.ReadVarInt(&r)
	if err != nil {
		return nil, err
	}

	txs, err = transactions.FromReader2(&r, lTxs, txs)
	if err != nil {
		return nil, err
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
	rewardBytes := make([]byte, 32)
	binary.LittleEndian.PutUint64(rewardBytes[:32], config.GeneratorReward)

	// Add the Block Generator reward
	output := &transactions.Output{}
	output.DestKey = P.Bytes()
	// Commitment field in coinbase tx represents the reward
	output.Commitment = rewardBytes

	tx.AddReward(output)

	// TODO: Optional here could be to verify if the reward is spendable by the generator wallet.
	// This could be achieved with a request to dusk-wallet

	return tx, nil
}

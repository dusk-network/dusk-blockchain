package core

import (
	"encoding/binary"
	"math/rand"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

// Generate will check if a bid's score is under the treshold, then constructs
// a candidate block and broadcasts it.
func (b *Blockchain) Generate(secret []byte) error {
	// Get tau here from config later
	tau := uint64(200)

	// Generate Y from function parameters
	if _, err := generateY(b.bidWeight, b.lastHeader.Seed, secret); err != nil {
		return err
	}

	// Gamma function (change out later)
	score := rand.Uint64()
	if score >= tau {
		return nil
	}

	candidateBlock := payload.NewBlock()
	if err := candidateBlock.SetPrevBlock(b.lastHeader); err != nil {
		return err
	}

	// Set seed with BLS
	if err := candidateBlock.SetSeed(b.lastHeader.Seed, b.BLSSecretKey); err != nil {
		return err
	}

	candidateBlock.SetTime(time.Now().Unix())

	// Generate coinbase/reward beforehand
	// Coinbase is still not decided
	txs := b.memPool.GetAllTxs()
	for _, tx := range txs {
		candidateBlock.AddTx(tx)
	}

	if err := candidateBlock.SetRoot(); err != nil {
		return err
	}

	if err := candidateBlock.SetHash(); err != nil {
		return err
	}

	// Sign with BLS
	sig, err := bls.Sign(b.BLSSecretKey, candidateBlock.Header.Hash)
	if err != nil {
		return err
	}

	if _, err := payload.NewMsgCandidate(candidateBlock.Header.Hash, sig, b.BLSPubKey); err != nil {
		return err
	}

	payload.NewMsgBlock(candidateBlock)

	// Propagate msgs

	return nil
}

func generateY(bid uint64, prevSeed, secret []byte) ([]byte, error) {
	bidAndSecret := make([]byte, 40)
	binary.LittleEndian.PutUint64(bidAndSecret, bid)
	bidAndSecret = append(bidAndSecret, secret...)

	firstHash, err := hash.Sha3256(bidAndSecret)
	if err != nil {
		return nil, err
	}

	seedAndFirstHash := append(secret, firstHash...)
	secondHash, err := hash.Sha3256(seedAndFirstHash)
	if err != nil {
		return nil, err
	}

	bidAndSecondHash := make([]byte, 40)
	binary.LittleEndian.PutUint64(bidAndSecret, bid)
	bidAndSecondHash = append(bidAndSecondHash, secondHash...)
	return hash.Sha3256(bidAndSecondHash)
}

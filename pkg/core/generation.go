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
func (b *Blockchain) Generate(bid uint64, secret []byte) (bool, error) {
	// Get tau here from config later
	tau := uint64(200)

	// Generate BLS keys
	blsPub, _, err := bls.GenKeyPair(nil)
	if err != nil {
		return false, err
	}

	// Get previous header
	prevHeader, err := b.GetLatestHeader()
	if err != nil {
		return false, err
	}

	// Generate Y from function parameters
	if _, err := generateY(bid, prevHeader.Seed, secret); err != nil {
		return false, err
	}

	// Gamma function (change out later)
	score := rand.Uint64()
	if score >= tau {
		return false, nil
	}

	candidateBlock := payload.NewBlock()
	if err := candidateBlock.SetPrevBlock(prevHeader); err != nil {
		return false, err
	}

	// Set seed with BLS once completed

	candidateBlock.SetTime(time.Now().Unix())

	// Generate coinbase/reward beforehand
	// Coinbase is still not decided
	txs := b.memPool.GetAllTxs()
	for _, tx := range txs {
		candidateBlock.AddTx(tx)
	}

	if err := candidateBlock.SetRoot(); err != nil {
		return false, err
	}

	if err := candidateBlock.SetHash(); err != nil {
		return false, err
	}

	// Set to sign with BLS once completed
	sig, err := hash.Sha3256(candidateBlock.Header.Hash)
	if err != nil {
		return false, err
	}

	binPub, err := blsPub.MarshalBinary()
	if err != nil {
		return false, err
	}

	// Change binPub[:32] when compressed BLS signatures are implemented
	if _, err := payload.NewMsgCandidate(candidateBlock.Header.Hash, sig, binPub[:32]); err != nil {
		return false, err
	}

	payload.NewMsgBlock(candidateBlock)

	// Propagate msgs

	// Successfully propagated block candidate
	return true, nil
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

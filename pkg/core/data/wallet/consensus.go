package wallet

import (
	"encoding/binary"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	zkproof "github.com/dusk-network/dusk-zkproof"
	"golang.org/x/crypto/sha3"
)

func generateConsensusKeys(seed []byte) (key.ConsensusKeys, error) {
	// Consensus keys require >80 bytes of seed, so we will hash seed twice and concatenate
	// both hashes to get 128 bytes

	seedHash := sha3.Sum512(seed)
	secondSeedHash := sha3.Sum512(seedHash[:])

	consensusSeed := append(seedHash[:], secondSeedHash[:]...)

	return key.NewConsensusKeysFromBytes(consensusSeed)
}

func generateM(PrivateSpend []byte, index uint32) []byte {

	// To make K deterministic
	// We will calculate K = PrivateSpend || Index
	// Index is the number of Bidding transactions that has
	// been initiated. This information should be available to the wallet
	// M = H(K)

	numBidTxsSeen := make([]byte, 4)
	binary.BigEndian.PutUint32(numBidTxsSeen, index)

	KBytes := append(PrivateSpend, numBidTxsSeen...)

	// Encode K as a ristretto Scalar
	var k ristretto.Scalar
	k.Derive(KBytes)

	m := zkproof.CalculateM(k)
	return m.Bytes()
}

// ReconstructK reconstructs the secret K from a scalar
// Deprecated: we are not using ristretto anymore as we moved to PLONK on
// jubjub
func (w *Wallet) ReconstructK() (ristretto.Scalar, error) {
	zeroPadding := make([]byte, 4)
	privSpend, err := w.PrivateSpend()
	if err != nil {
		return ristretto.Scalar{}, err
	}

	kBytes := append(privSpend, zeroPadding...)
	var k ristretto.Scalar
	k.Derive(kBytes)
	return k, nil
}

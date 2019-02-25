package sortition

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/big"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

// CreateCommittee will run the deterministic sortition function, which determines
// who will be in the committee for a given step.
func CreateCommittee(round, totalWeight uint64, step uint8, committee [][]byte,
	stakes map[string]uint64) ([][]byte, error) {
	var currentCommittee [][]byte
	W := new(big.Int).SetUint64(totalWeight)
	size := len(committee)
	if size > 50 {
		size = 50
	}

	for i := 0; i < size; i++ {
		// Create message to hash
		msg := make([]byte, 10)
		binary.LittleEndian.PutUint64(msg[:8], round)
		msg = append(msg, byte(step))
		msg = append(msg, byte(i))

		// Hash message
		hash, err := hash.Sha3256(msg)
		if err != nil {
			return nil, err
		}

		// Generate score
		hashNum := new(big.Int).SetBytes(hash)
		scoreNum := new(big.Int).Mod(hashNum, W)
		score := scoreNum.Uint64()

		// Walk through the committee set and keep deducting until we reach zero
		for _, pk := range committee {
			pkStr := hex.EncodeToString(pk)
			if stakes[pkStr] >= score {
				currentCommittee = append(currentCommittee, pk)
				break
			}

			score -= stakes[pkStr]
		}
	}

	return currentCommittee, nil
}

// Verify will check if a public key is included in a passed committee, and return
// how many times they appear.
func Verify(committee [][]byte, pk []byte) uint8 {
	// Check if this node is eligible, and how many votes they get
	votes := uint8(0)
	for _, pkB := range committee {
		if bytes.Equal(pkB, pk) {
			votes++
		}
	}

	return votes
}

package sortition

import (
	"encoding/binary"
	"math/big"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

// CreateCommittee will run the deterministic sortition function, which determines
// who will be in the committee for a given step.
func CreateCommittee(round, totalWeight uint64, step uint32, committee *user.Committee,
	stakes map[string]uint64) (map[string]uint8, error) {
	currentCommittee := make(map[string]uint8)
	W := new(big.Int).SetUint64(totalWeight)
	size := len(*committee)
	if size > 50 {
		size = 50
	}

	for i := uint8(0); i < uint8(size); i++ {
		// Create message to hash
		msg := make([]byte, 12)
		binary.LittleEndian.PutUint64(msg[:8], round)
		binary.LittleEndian.PutUint32(msg[8:], step)
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
		for _, m := range *committee {
			if stakes[m.String()] >= score {
				currentCommittee[m.String()]++
				break
			}

			score -= stakes[m.String()]
		}
	}

	return currentCommittee, nil
}

// SetCommittee will set the committee for the given step
func SetCommittee(ctx *user.Context, step uint32) error {
	currentCommittee, err := CreateCommittee(ctx.Round, ctx.W, step,
		ctx.Committee, ctx.NodeWeights)
	if err != nil {
		return err
	}

	ctx.CurrentCommittee = currentCommittee
	return nil
}

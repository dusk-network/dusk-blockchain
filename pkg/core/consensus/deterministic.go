package consensus

import (
	"encoding/binary"
	"encoding/hex"
	"math/big"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

// Deterministic will run the deterministic sortition function.
func Deterministic(ctx *Context) error {
	for i := uint8(0); i < committeeSize; i++ {
		// Create message to hash
		msg := make([]byte, 10)
		binary.LittleEndian.PutUint64(msg[:8], ctx.Round)
		msg = append(msg, byte(ctx.Step))
		msg = append(msg, byte(i))

		// Hash message
		hash, err := hash.Sha3256(msg)
		if err != nil {
			return err
		}

		// Generate score
		W := new(big.Int).SetUint64(ctx.W)
		hashNum := new(big.Int).SetBytes(hash)
		scoreNum := new(big.Int).Mod(hashNum, W)
		score := scoreNum.Uint64()

		// Walk through the committee set and keep deducting until we reach zero
		for _, pk := range ctx.Committee {
			pkStr := hex.EncodeToString(pk)
			if ctx.NodeWeights[pkStr] >= score {
				ctx.CurrentCommittee[i] = pk
				break
			}

			score -= ctx.NodeWeights[pkStr]
		}
	}

	return nil
}

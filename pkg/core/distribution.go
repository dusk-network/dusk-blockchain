package core

import (
	"math/big"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gonum.org/v1/gonum/stat/distuv"
)

// calcNormDist is used for block reduction. It uses a normal
// distribution function.
func calcNormDist(threshold float64, weight, totalWeight uint64, score *bls.Sig) int {
	// Calculate probability (sigma)
	p := threshold / float64(totalWeight)

	// Calculate score divided by 2^256
	var lenHash, _ = new(big.Int).SetString("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0)

	// Set from marshaled score later
	scoreNum := new(big.Int).SetBytes([]byte{0, 0})
	target, _ := new(big.Rat).SetFrac(scoreNum, lenHash).Float64()

	j := float64(0.0)
	for {
		dist := distuv.Normal{
			Mu:    float64(weight),
			Sigma: p,
		}

		p1 := dist.Prob(j)
		p2 := dist.Prob(j + 1.0)
		if p1 > target && target > p2 {
			break
		}

		j += 1.0
	}

	return int(j)
}

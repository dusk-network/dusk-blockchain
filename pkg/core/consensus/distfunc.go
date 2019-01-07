package consensus

import (
	"encoding/binary"
	"math"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

// distfunc is the function used to generate the score using gamma distribution
// XXX: Ask Jules where is normal distribution used?

// GenerateScore generates the score Q, for the block generation hase
// Is this meant to be deterministic on d and Y?
func GenerateScore(d uint64, Y []byte) uint64 {

	// XXX: For now we just calculate the pdf at a value x

	yHashed, _ := hash.Sha3256(Y)
	y := binary.LittleEndian.Uint64(yHashed)

	pdfA := logProb(float64(d))
	pdfB := logProb(float64(y))

	if pdfA < pdfB {
		return uint64(pdfB)
	}

	return uint64(pdfA) // very basic affine transformation?
}

// LogProb computes the natural logarithm of the value of the probability
// density function at x.
// Copied from GoNum
func logProb(x float64) float64 {
	if x <= 0 {
		return math.Inf(-1)
	}
	alpha := 0.45
	beta := 0.76

	lg, _ := math.Lgamma(alpha)
	return alpha*math.Log(beta) - lg + (alpha-1)*math.Log(x) - beta*x
}

// At the moment we are using Gamma distribution
// Y must therefore be casted as a uint, or we must ensure that when we mix Y with d, that it is >=0
// XXX: Can this be replaced with a discrete function? since range is [0, 2^64).. We could just use ceil, as I think it's fine for two nodes to have the same score

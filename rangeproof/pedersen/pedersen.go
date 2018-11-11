package pedersen

import (
	generator "github.com/toghrulmaharramov/dusk-go/rangeproof/generators"
	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

type Pedersen struct {
	H          ristretto.Point
	G          ristretto.Point
	BaseVector *generator.Generator
	GenData    []byte
}

// New will setup the H, G, BaseVector and Blinding Vector values
// returning a Pedersen
// genData is the byte slice, that will be used
// to form the unique set of generators
func New(genData []byte) *Pedersen {
	gen := generator.New(genData)

	// XXX: Use H and G in the constants, so we do not keep recalculating

	G := ristretto.Point{}
	G.SetBase()

	H := ristretto.Point{}
	H.DeriveDalek(G.Bytes()) // XXX: Is there consistent usage of Derive/DeriveDalek?

	return &Pedersen{
		H:          H,
		G:          G,
		BaseVector: gen,
		GenData:    genData,
	}
}

// Commitment represents a Pedersen Commitment
// storing the value and the random blinding factor
type Commitment struct {
	// Value is the point which has been commited to
	Value ristretto.Point
	// blinding factor is the blinding scalar.
	// Note that n vectors have 1 blinding factor
	BlindingFactor ristretto.Scalar
}

// Add will add commm to c, but keep the original blinding factor of c
func (c *Commitment) Add(comm Commitment) {

	if c.BlindingFactor.IsNonZeroI() == 0 { // 0 means blinding factor is zero

		// If we get here, then user has used the commitment struct without setting a value
		// so set value to zero and generate a random blindig factor

		s := ristretto.Scalar{}
		s.Rand()

		c.Value.SetZero()
	}

	c.Value.Add(&c.Value, &comm.Value)

}

// P = aG + bH where a is the blinding factor
func (p *Pedersen) CommitToScalar(b ristretto.Scalar) Commitment {
	// Generate random blinding factor
	a := ristretto.Scalar{}
	a.Rand()

	aG := ristretto.Point{}
	aG.ScalarMult(&p.G, &a)

	bH := ristretto.Point{}
	bH.ScalarMult(&p.H, &b)

	// aG+bH
	aGbH := ristretto.Point{}
	aGbH.Add(&aG, &bH)

	return Commitment{
		Value:          aGbH,
		BlindingFactor: a,
	}
}

// P = aG + <b_1*H_1+b_2*H_2+b_3*H_3...b_i*H_i> where a is the blinding factor
func (p *Pedersen) CommitToVector(vec []ristretto.Scalar) Commitment {
	// Generate random blinding factor
	a := ristretto.Scalar{}
	a.Rand()

	if len(p.BaseVector.Bases) < len(vec) {
		diff := len(p.BaseVector.Bases) - len(vec)
		p.BaseVector.Compute(uint8(diff))
		// len of vector to commit should be equal or less than the number of precompued generators
	}

	sum := ristretto.Point{}
	sum.SetZero()

	for i := range vec {
		b_i := vec[i]
		H_i := p.BaseVector.Bases[i] // XXX: Note that we do not increment the underlying counter in generator.
		// This is to be used per vector for which we need a different set of generators.

		// H_i * b_i
		product := ristretto.Point{}
		product.ScalarMult(&H_i, &b_i)

		sum.Add(&sum, &product)

	}

	return Commitment{
		Value:          sum,
		BlindingFactor: a,
	}
}

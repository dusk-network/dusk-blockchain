package pedersen

import (
	generator "gitlab.dusk.network/dusk-core/dusk-go/rangeproof/generators"
	"gitlab.dusk.network/dusk-core/dusk-go/ristretto"
)

type Pedersen struct {
	BaseVector *generator.Generator
	GenData    []byte
}

// New will setup the BaseVector
// returning a Pedersen struct
// genData is the byte slice, that will be used
// to form the unique set of generators
func New(genData []byte) *Pedersen {
	gen := generator.New(genData)

	return &Pedersen{
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

func (p *Pedersen) commitToScalars(blind *ristretto.Scalar, scalars ...ristretto.Scalar) ristretto.Point {

	n := len(scalars)

	if blind != nil {
		n++
		// prepend blinding factor to slice of scalars
		scalars = append([]ristretto.Scalar{*blind}, scalars...)
	}

	if len(p.BaseVector.Bases) < n {

		diff := n - len(p.BaseVector.Bases)

		p.BaseVector.Compute(uint32(diff))
		// num of scalars to commit should be equal or less than the number of precompued generators
	}

	var sum ristretto.Point
	sum.SetZero()

	for i := 0; i < n; i++ {

		var bi ristretto.Scalar

		// commitment for the blinding factor
		bi = scalars[i]

		// for each scalar, we commit to a point on the main generator, We do not change generator here
		Hi := p.BaseVector.Bases[i]

		// H_i * b_i
		product := ristretto.Point{}
		product.ScalarMult(&Hi, &bi)

		sum.Add(&sum, &product)
	}

	return sum
}

func (p *Pedersen) CommitToScalars(scalars ...ristretto.Scalar) Commitment {

	// Generate random blinding factor
	blind := ristretto.Scalar{}
	blind.Rand()

	sum := p.commitToScalars(&blind, scalars...)

	return Commitment{
		Value:          sum,
		BlindingFactor: blind,
	}
}

func (p *Pedersen) CommitToVectors(vectors ...[]ristretto.Scalar) Commitment {

	// Generate random blinding factor
	blind := ristretto.Scalar{}
	blind.Rand()

	// For each vector, we can use the commitToScalars, because a vector is just a slice of scalars

	var sum ristretto.Point
	sum.SetZero()

	for i, vector := range vectors {
		if i == 0 {
			// Commit to vector + blinding factor

			commit := p.commitToScalars(&blind, vector...)
			sum.Add(&sum, &commit)
		} else {

			// new generator -- XXX: we could use a hashing function here instead
			// XXX: Does this clash with generators iterate() function?
			genData := append(p.GenData, uint8(i))
			ped2 := New(genData)

			commit := ped2.commitToScalars(nil, vector...)
			sum.Add(&sum, &commit)
		}
	}

	return Commitment{
		Value:          sum,
		BlindingFactor: blind,
	}
}

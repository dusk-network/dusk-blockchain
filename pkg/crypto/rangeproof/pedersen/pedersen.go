package pedersen

import (
	generator "gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/generators"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/ristretto"
)

type Pedersen struct {
	BaseVector *generator.Generator
	GenData    []byte
	BlindPoint ristretto.Point // This point will be used to Commit the blinders for the blinding scalars
	BasePoint  ristretto.Point // This point will be used to Commit the amounts for the amount scalars
}

// New will setup the BaseVector
// returning a Pedersen struct
// genData is the byte slice, that will be used
// to form the unique set of generators
func New(genData []byte) *Pedersen {
	gen := generator.New(genData)

	var blindPoint ristretto.Point
	var basePoint ristretto.Point

	blindPoint.Derive([]byte("blindPoint"))
	basePoint.SetBase()

	return &Pedersen{
		BaseVector: gen,
		GenData:    genData,
		BlindPoint: blindPoint,
		BasePoint:  basePoint,
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

func (p *Pedersen) commitToScalars(blind *ristretto.Scalar, scalars ...ristretto.Scalar) ristretto.Point {

	n := len(scalars)

	var sum ristretto.Point
	sum.SetZero()

	if blind != nil {

		var blindPoint ristretto.Point
		blindPoint.ScalarMult(&p.BlindPoint, blind)
		sum.Add(&sum, &blindPoint)
	}

	if len(p.BaseVector.Bases) < n {

		diff := n - len(p.BaseVector.Bases)

		p.BaseVector.Compute(uint32(diff))
		// num of scalars to commit should be equal or less than the number of precomputed generators
	}

	for i := 0; i < n; i++ {

		bi := scalars[i]

		Hi := p.BaseVector.Bases[i]

		// H_i * b_i
		product := ristretto.Point{}
		product.ScalarMult(&Hi, &bi)

		sum.Add(&sum, &product)
	}

	return sum
}

// CommitToScalar generates a Commitment to a scalar v, s.t. V = v * Base + blind * BlindingPoint
func (p *Pedersen) CommitToScalar(v ristretto.Scalar) Commitment {

	// generate random blinder
	blind := ristretto.Scalar{}
	blind.Rand()

	// v * Base
	var vBase ristretto.Point
	vBase.ScalarMult(&p.BasePoint, &v)
	// blind * BlindPoint
	var blindPoint ristretto.Point
	blindPoint.ScalarMult(&p.BlindPoint, &blind)

	var sum ristretto.Point
	sum.SetZero()
	sum.Add(&vBase, &blindPoint)

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

			// new generator -- XXX: we could use a hashing function here instead of appending i to it?
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

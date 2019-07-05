package pedersen

import (
	"encoding/binary"
	"errors"
	"io"

	ristretto "github.com/bwesterb/go-ristretto"
	generator "gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/generators"
)

// Pedersen represents a pedersen struct which holds
// the necessary information to commit a vector or a scalar to a point
type Pedersen struct {
	BaseVector *generator.Generator
	GenData    []byte
	BlindPoint ristretto.Point
	BasePoint  ristretto.Point
}

// New will setup the BaseVector
// returning a Pedersen struct
// genData is the byte slice, that will be used
// to form the unique set of generators
func New(genData []byte) *Pedersen {
	gen := generator.New(genData)

	var blindPoint ristretto.Point
	var basePoint ristretto.Point

	basePoint.Derive([]byte("blindPoint"))
	blindPoint.SetBase()

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

func (c *Commitment) Encode(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, c.Value.Bytes())
}

func EncodeCommitments(w io.Writer, comms []Commitment) error {
	lenV := uint32(len(comms))
	err := binary.Write(w, binary.BigEndian, lenV)
	if err != nil {
		return err
	}
	for i := range comms {
		err := comms[i].Encode(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Commitment) Decode(r io.Reader) error {
	if c == nil {
		return errors.New("struct is nil")
	}

	var cBytes [32]byte
	err := binary.Read(r, binary.BigEndian, &cBytes)
	if err != nil {
		return err
	}
	ok := c.Value.SetBytes(&cBytes)
	if !ok {
		return errors.New("could not set bytes for commitment, not an encodable point")
	}
	return nil
}

func DecodeCommitments(r io.Reader) ([]Commitment, error) {

	var lenV uint32
	err := binary.Read(r, binary.BigEndian, &lenV)
	if err != nil {
		return nil, err
	}

	comms := make([]Commitment, lenV)

	for i := uint32(0); i < lenV; i++ {
		err := comms[i].Decode(r)
		if err != nil {
			return nil, err
		}
	}

	return comms, nil
}

func (c *Commitment) EqualValue(other Commitment) bool {
	return c.Value.Equals(&other.Value)
}

func (c *Commitment) Equals(other Commitment) bool {
	return c.EqualValue(other) && c.BlindingFactor.Equals(&other.BlindingFactor)
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

// CommitToVectors will take n set of vectors and form a commitment to them s.t.
// V = aH + <v1, G1> + <v2, G2> + <v3, G3>
// where a is a scalar, v1 is a vector of scalars, and G1 is a vector of points
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

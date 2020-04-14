package key

import (
	"errors"

	ristretto "github.com/bwesterb/go-ristretto"
)

// PublicSpend represents the public spend key
type PublicSpend ristretto.Point

func pubSpendFromBytes(byt [32]byte) (*PublicSpend, error) {

	var point ristretto.Point
	ok := point.SetBytes(&byt)
	if !ok {
		return nil, errors.New("could not set Public Spend Bytes")
	}

	pubSpend := PublicSpend(point)
	return &pubSpend, nil
}

// ScalarMult performs a scalar multiplication between this PublicSpend and
// ristretto scalar
func (ps PublicSpend) ScalarMult(s ristretto.Scalar) PublicSpend {
	var p ristretto.Point
	p.ScalarMult(ps.point(), &s)
	return PublicSpend(p)
}

func (ps PublicSpend) String() string {
	return ps.point().String()
}

func (ps PublicSpend) point() *ristretto.Point {
	p := (ristretto.Point)(ps)
	return &p
}

// Bytes returns the byte representation of the PublicSpend point
func (ps PublicSpend) Bytes() []byte {
	return ps.point().Bytes()
}

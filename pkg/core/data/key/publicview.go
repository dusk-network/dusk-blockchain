package key

import (
	"errors"

	ristretto "github.com/bwesterb/go-ristretto"
)

//PublicView is the public view key
type PublicView ristretto.Point

func (pv PublicView) String() string {
	return pv.point().String()
}

// Bytes representation of the PublicView ristretto point
func (pv PublicView) Bytes() []byte {
	return pv.point().Bytes()
}

func (pv PublicView) point() *ristretto.Point {
	p := (ristretto.Point)(pv)
	return &p
}

//ScalarMult performs a scalar multiplication of the ristretto points within
//this PublicView
func (pv PublicView) ScalarMult(s ristretto.Scalar) PublicView {
	var p ristretto.Point
	p.ScalarMult(pv.point(), &s)
	return PublicView(p)
}

func pubViewFromBytes(byt [32]byte) (*PublicView, error) {

	var point ristretto.Point
	ok := point.SetBytes(&byt)
	if !ok {
		return nil, errors.New("could not set Public View Bytes")
	}

	pubView := PublicView(point)
	return &pubView, nil
}

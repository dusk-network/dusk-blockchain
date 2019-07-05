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

func (pv PublicView) Bytes() []byte {
	return pv.point().Bytes()
}

func (pv PublicView) point() *ristretto.Point {
	p := (ristretto.Point)(pv)
	return &p
}

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

	var pubView PublicView
	pubView = PublicView(point)

	return &pubView, nil
}

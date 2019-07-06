package key

import ristretto "github.com/bwesterb/go-ristretto"

//PrivateView represents the private view key
type PrivateView ristretto.Scalar

// PublicView returns a public view key from a private view key
func (pv PrivateView) PublicView() *PublicView {
	var publicViewKey PublicView

	var x ristretto.Point
	x.ScalarMultBase(pv.scalar())

	publicViewKey = PublicView(x)

	return &publicViewKey
}

func (pv PrivateView) scalar() *ristretto.Scalar {
	s := (ristretto.Scalar)(pv)
	return &s
}

func (pv PrivateView) Bytes() []byte {
	return pv.scalar().Bytes()
}

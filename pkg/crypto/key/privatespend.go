package key

import ristretto "github.com/bwesterb/go-ristretto"

//PrivateSpend represents a private spend key
type PrivateSpend ristretto.Scalar

func newSpendFromSeed(seed []byte) *PrivateSpend {

	var privateSpend PrivateSpend

	var x ristretto.Scalar
	x.Derive(seed)

	privateSpend = PrivateSpend(x)
	return &privateSpend
}

// PrivateView converts a private spend key to a private view key
func (pr PrivateSpend) PrivateView() *PrivateView {
	var pvScalar ristretto.Scalar
	pvScalar.Derive(pr.Bytes())

	pv := PrivateView(pvScalar)
	return &pv
}

// PublicSpend converts a private spend key to a public spend key
func (pr PrivateSpend) PublicSpend() *PublicSpend {
	var publicSpendKey PublicSpend

	var x ristretto.Point
	x.ScalarMultBase(pr.scalar())

	publicSpendKey = PublicSpend(x)

	return &publicSpendKey
}

func (pr PrivateSpend) scalar() *ristretto.Scalar {
	s := (ristretto.Scalar)(pr)
	return &s
}

func (pr PrivateSpend) Bytes() []byte {
	return pr.scalar().Bytes()
}

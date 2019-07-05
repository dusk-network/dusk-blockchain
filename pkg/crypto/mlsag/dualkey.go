package mlsag

import (
	"errors"

	ristretto "github.com/bwesterb/go-ristretto"
)

// DualKey is a specific instantiation of mlsag where the second key is
// a commitment to zero
type DualKey struct {
	Proof
	dualkeys []ristretto.Scalar
}

func NewDualKey() *DualKey {
	return &DualKey{
		Proof:    Proof{},
		dualkeys: make([]ristretto.Scalar, 2),
	}
}

func (d *DualKey) SetPrimaryKey(key ristretto.Scalar) ristretto.Point {
	d.dualkeys[0] = key
	return privKeyToPubKey(key)
}

func (d *DualKey) SetCommToZero(key ristretto.Scalar) ristretto.Point {
	d.dualkeys[1] = key
	return privKeyToPubKey(key)
}

func (d *DualKey) SetMsg(msg []byte) {
	d.Proof.msg = msg
}

// SubCommToZero subtracts p from every point from the second public key
// in the matrix of decoy pubkeys
func (d *DualKey) SubCommToZero(p ristretto.Point) {
	for i := range d.pubKeysMatrix {
		commToZero := &d.pubKeysMatrix[i].keys[1]
		commToZero.Sub(commToZero, &p)
	}
}

func (d *DualKey) Prove() (*Signature, ristretto.Point, error) {

	if (d.dualkeys[0].IsNonZeroI() == 0) || (d.dualkeys[1].IsNonZeroI() == 0) {
		return nil, ristretto.Point{}, errors.New("primary key or commitment to zero cannot be zero")
	}

	d.AddSecret(d.dualkeys[0])
	d.AddSecret(d.dualkeys[1])

	sig, keyimage, err := d.prove(true)
	if err != nil {
		return nil, ristretto.Point{}, err
	}
	if len(keyimage) != 1 {
		return nil, ristretto.Point{}, errors.New("dual key mlsag must only contain one key image")
	}
	return sig, keyimage[0], err
}

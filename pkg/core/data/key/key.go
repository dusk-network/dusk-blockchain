package key

import (
	"errors"

	ristretto "github.com/bwesterb/go-ristretto"
)

// Key represents the private/public spend/view key pairs
type Key struct {
	privKey *PrivateKey
	pubKey  *PublicKey
}

// NewKeyPair returns a pair of public and private keys
func NewKeyPair(seed []byte) *Key {
	privKey := newPrivateKey(seed)
	pubKey := privKey.publicKey()

	return &Key{
		privKey,
		pubKey,
	}
}

// PublicKey returns the corresponding public key pair
func (k Key) PublicKey() *PublicKey {
	if k.pubKey != nil {
		return k.pubKey
	}

	k.pubKey = &PublicKey{
		k.privKey.privSpend.PublicSpend(),
		k.privKey.privView.PublicView(),
	}

	return k.pubKey
}

// PrivateView returns the private view key
// Used when decrypting amounts and masks
func (k Key) PrivateView() (*PrivateView, error) {

	if k.privKey.privView == nil {
		return nil, errors.New("private view key is nil")
	}
	return k.privKey.privView, nil
}

// PrivateSpend returns the private spend key
func (k Key) PrivateSpend() (*PrivateSpend, error) {

	if k.privKey.privSpend == nil {
		return nil, errors.New("private spend key is nil")
	}
	return k.privKey.privSpend, nil
}

// DidReceiveTx takes P the stealthAddress/ one time pubkey
// and the tx pubkey R
// checks whether the tx was intended for the key assosciated
func (k *Key) DidReceiveTx(R ristretto.Point, stealth StealthAddress, index uint32) (*ristretto.Scalar, bool) {

	pubKey := k.PublicKey()

	var Dprime ristretto.Point
	Dprime.ScalarMult(&R, k.privKey.privView.scalar())

	var fprime ristretto.Scalar
	DprimeIndex := concatSlice(Dprime.Bytes(), uint32ToBytes(index))
	fprime.Derive(DprimeIndex)

	var Fprime ristretto.Point
	Fprime.ScalarMultBase(&fprime)

	Pprime := Fprime.Add(pubKey.PubSpend.point(), &Fprime)

	if stealth.P.Equals(Pprime) {
		x := fprime.Add(&fprime, k.privKey.privSpend.scalar())
		return x, true
	}
	return nil, false
}

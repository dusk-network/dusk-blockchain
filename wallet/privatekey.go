package wallet

import (
	"bytes"
	cr "crypto"
	"errors"
	"math/big"

	"github.com/toghrulmaharramov/dusk-go/crypto"

	"golang.org/x/crypto/ed25519"
)

var (
	//PrivKeyPrefix is the prefix needed to produce a WIF
	// that starts with DUSKpriv
	PrivKeyPrefix = big.NewInt((int64(0x265CC557D4C1F1)))
)

// PrivateKey wraps around the native ed25519 private key
// with modified methods.
type PrivateKey struct {
	ed25519.PrivateKey
	PubKey
}

// Public overrides the native ed25519 method
// to return a PubKey
func (p *PrivateKey) Public() *PubKey {
	return &p.PubKey
}

// WIF refers to the Wallet Import Format for the private Key
func (p *PrivateKey) WIF() (string, error) {
	if len(p.PrivateKey) != 32 {
		return "", errors.New("PrivKey length does not equal 32")
	}
	return crypto.KeyToAddress(PrivKeyPrefix, p.PrivateKey, 1)
}

// Sign Wraps around the native ed25519 function
// adding extra functionality. As per the docs, SHA3 cannot be used
// to pre-hash
func (p *PrivateKey) Sign(message []byte) (sig []byte, err error) {

	en, err := crypto.RandEntropy(33)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(en)

	return p.PrivateKey.Sign(r, message, cr.Hash(0))
}

// NewPrivateKey returns a new PrivateKey
func NewPrivateKey() (*PrivateKey, error) {

	entropy, err := crypto.RandEntropy(32)

	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(entropy)

	pub, priv, err := ed25519.GenerateKey(r)

	return &PrivateKey{
		priv,
		PubKey{
			pub,
		},
	}, err
}

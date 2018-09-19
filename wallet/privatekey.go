package wallet

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"errors"
	"strconv"

	"golang.org/x/crypto/ed25519"
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

// Sign Wraps around the native ed25519 function
// adding extra functionality. As per the docs, SHA3 cannot be used
// to pre-hash
func (p *PrivateKey) Sign(message []byte) (sig []byte, err error) {

	en, err := randEntropy(33)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(en)

	return p.PrivateKey.Sign(r, message, crypto.Hash(0))
}

// NewPrivateKey returns a new PrivateKey
func NewPrivateKey() (*PrivateKey, error) {

	entropy, err := randEntropy(32)

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

func randEntropy(n int) ([]byte, error) {

	if n < 32 {
		return nil, errors.New("n should be more than 32 bytes")
	}

	b := make([]byte, n)
	a, err := rand.Read(b)

	if err != nil {
		return nil, errors.New("Error generating entropy " + err.Error())
	}
	if a != n {
		return nil, errors.New("Error expected to read" + strconv.Itoa(n) + " bytes instead read " + strconv.Itoa(a) + " bytes")
	}
	return b, nil
}

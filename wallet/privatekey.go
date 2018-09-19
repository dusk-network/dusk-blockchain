package wallet

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"errors"
	"math/big"
	"strconv"

	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
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
	return KeyToAddress(PrivKeyPrefix, p.PrivateKey, 1)
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

func Checksum(data []byte) ([]byte, error) {
	hash, err := DoubleSha3256(data)
	if err != nil {
		return nil, err
	}
	return hash[:4], nil
}

func Sha3256(data []byte) ([]byte, error) {
	hasher := sha3.New256()
	hasher.Reset()
	_, err := hasher.Write(data)
	if err != nil {
		return nil, err
	}
	return hasher.Sum(nil), nil
}

func DoubleSha3256(data []byte) ([]byte, error) {
	h, err := Sha3256(data)
	if err != nil {
		return nil, err
	}
	hash, err := Sha3256(h)
	if err != nil {
		return nil, err
	}
	return hash, nil
}

package user

import (
	"bytes"
	"io"

	"github.com/dusk-network/dusk-crypto/bls"
	"golang.org/x/crypto/ed25519"
)

// Keys are the keys used during consensus
type Keys struct {
	BLSPubKey      *bls.PublicKey
	BLSPubKeyBytes []byte
	BLSSecretKey   *bls.SecretKey
	EdPubKey       *ed25519.PublicKey
	EdPubKeyBytes  []byte
	EdSecretKey    *ed25519.PrivateKey
}

// NewRandKeys will generate and return new bls and ed25519
// keys to be used in consensus
func NewRandKeys() (Keys, error) {
	return NewKeysFromReader(nil)
}

// NewKeysFromReader takes a reader and uses it as a seed for generating random user.Keys
func NewKeysFromReader(r io.Reader) (Keys, error) {
	blsPub, blsPriv, err := bls.GenKeyPair(r)
	if err != nil {
		return Keys{}, err
	}

	edPub, edPriv, err := ed25519.GenerateKey(r)
	if err != nil {
		return Keys{}, err
	}

	return Keys{
		BLSPubKey:      blsPub,
		BLSSecretKey:   blsPriv,
		EdPubKey:       &edPub,
		EdPubKeyBytes:  []byte(edPub),
		EdSecretKey:    &edPriv,
		BLSPubKeyBytes: blsPub.Marshal(),
	}, nil
}

// NewKeysFromBytes gets an array of bytes as seed for the pseudo-random generation of user.Keys
func NewKeysFromBytes(seed []byte) (Keys, error) {

	r := bytes.NewBuffer(seed)
	r.Grow(len(seed))
	blsPub, blsPriv, err := bls.GenKeyPair(r)
	if err != nil {
		return Keys{}, err
	}

	r = bytes.NewBuffer(seed)
	r.Grow(len(seed))
	edPub, edPriv, err := ed25519.GenerateKey(r)
	if err != nil {
		return Keys{}, err
	}

	return Keys{
		BLSPubKey:      blsPub,
		BLSSecretKey:   blsPriv,
		EdPubKey:       &edPub,
		EdPubKeyBytes:  []byte(edPub),
		EdSecretKey:    &edPriv,
		BLSPubKeyBytes: blsPub.Marshal(),
	}, nil
}

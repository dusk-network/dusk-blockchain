package key

import (
	"bytes"
	"io"

	"github.com/dusk-network/dusk-crypto/bls"
)

// Keys are the keys used during consensus
type Keys struct {
	BLSPubKey      *bls.PublicKey
	BLSPubKeyBytes []byte
	BLSSecretKey   *bls.SecretKey
}

// NewRandKeys will generate and return new bls and ed25519
// keys to be used in consensus
func NewRandKeys() (Keys, error) {
	var keys Keys
	var err error
	for {
		keys, err = NewKeysFromReader(nil)
		if err == io.EOF {
			continue
		}
		break
	}

	return keys, err
}

// NewKeysFromReader takes a reader and uses it as a seed for generating random Keys
func NewKeysFromReader(r io.Reader) (Keys, error) {
	blsPub, blsPriv, err := bls.GenKeyPair(r)
	if err != nil {
		return Keys{}, err
	}

	return Keys{
		BLSPubKey:      blsPub,
		BLSSecretKey:   blsPriv,
		BLSPubKeyBytes: blsPub.Marshal(),
	}, nil
}

// NewKeysFromBytes gets an array of bytes as seed for the pseudo-random generation of Keys
func NewKeysFromBytes(seed []byte) (Keys, error) {

	r := bytes.NewBuffer(seed)
	r.Grow(len(seed))
	blsPub, blsPriv, err := bls.GenKeyPair(r)
	if err != nil {
		return Keys{}, err
	}

	return Keys{
		BLSPubKey:      blsPub,
		BLSSecretKey:   blsPriv,
		BLSPubKeyBytes: blsPub.Marshal(),
	}, nil
}

package user

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
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

// Clear removes the pointers to the Consensus Keys
//XXX: Would there be a situation, where we clear all references to the keys? Just to be safe? if no natural use-case, then lets remove it
func (k *Keys) Clear() {
	k.BLSPubKey = nil
	k.BLSSecretKey = nil
	k.EdPubKey = nil
	k.EdPubKeyBytes = nil
	k.EdSecretKey = nil
	k.BLSPubKeyBytes = nil
}

// NewRandKeys will generate and return new bls and ed25519
// keys to be used in consensus
func NewRandKeys() (*Keys, error) {
	blsPub, blsPriv, err := bls.GenKeyPair(nil)
	if err != nil {
		return nil, err
	}

	edPub, edPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}

	return &Keys{
		BLSPubKey:      blsPub,
		BLSSecretKey:   blsPriv,
		EdPubKey:       &edPub,
		EdPubKeyBytes:  []byte(edPub),
		EdSecretKey:    &edPriv,
		BLSPubKeyBytes: blsPub.Marshal(),
	}, nil
}

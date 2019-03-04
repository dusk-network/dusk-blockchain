package user

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"golang.org/x/crypto/ed25519"
)

// Keys are the keys used during consensus
type Keys struct {
	BLSPubKey    *bls.PublicKey
	BLSSecretKey *bls.SecretKey
	EdPubKey     *ed25519.PublicKey
	EdSecretKey  *ed25519.PrivateKey
}

// Clear removes the pointers to the Consensus Keys
//XXX: Would there be a situation, where we clear all references to the keys? Just to be safe? if no natural use-case, then lets remove it
func (k *Keys) Clear() {
	k.BLSPubKey = nil
	k.BLSSecretKey = nil
	k.EdPubKey = nil
	k.EdSecretKey = nil
}

// EdPubKeyBytes returns the byte slice of the ed25519 public key of this Keys
// struct.
func (k *Keys) EdPubKeyBytes() []byte {
	return []byte(*k.EdPubKey)
}

// // NewKeys will construct new Keys to be used during consensus
// // keys should already be generated here, just pass them through as params
// func NewKeys() (*Keys, error) {
// 	return &Keys{}, nil
// }

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
		BLSPubKey:    blsPub,
		BLSSecretKey: blsPriv,
		EdPubKey:     &edPub,
		EdSecretKey:  &edPriv,
	}, nil
}

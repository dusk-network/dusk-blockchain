package user

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/crypto"
	"github.com/stretchr/testify/assert"
)

func TestDeterministicKeyGen(t *testing.T) {
	seed, err := crypto.RandEntropy(128)
	assert.Nil(t, err)

	firstKeyPair, err := NewKeysFromBytes(seed)
	assert.Nil(t, err)

	secondKeyPair, err := NewKeysFromBytes(seed)
	assert.Nil(t, err)

	assert.Equal(t, firstKeyPair.BLSPubKeyBytes, secondKeyPair.BLSPubKeyBytes)
	assert.Equal(t, firstKeyPair.EdPubKeyBytes, secondKeyPair.EdPubKeyBytes)
	assert.Equal(t, firstKeyPair.BLSSecretKey, secondKeyPair.BLSSecretKey)
	assert.Equal(t, firstKeyPair.EdSecretKey, secondKeyPair.EdSecretKey)

}

package user

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestDeterministicKeyGen(t *testing.T) {
	seed, _ := crypto.RandEntropy(128)

	r := bytes.NewReader(seed)

	firstKeyPair, err := NewKeysFromSeed(r)
	assert.Nil(t, err)

	r = bytes.NewReader(seed)
	secondKeyPair, err := NewKeysFromSeed(r)
	assert.Nil(t, err)

	assert.Equal(t, firstKeyPair.BLSPubKeyBytes, secondKeyPair.BLSPubKeyBytes)
	assert.Equal(t, firstKeyPair.EdPubKeyBytes, secondKeyPair.EdPubKeyBytes)
	assert.Equal(t, firstKeyPair.BLSSecretKey, secondKeyPair.BLSSecretKey)
	assert.Equal(t, firstKeyPair.EdSecretKey, secondKeyPair.EdSecretKey)

}

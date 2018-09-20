package wallet

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/crypto"
)

func TestNewPrivKey(t *testing.T) {

	key := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	priv, _ := NewPrivateKey(key)

	expected := "0100000000000000000000000000000000000000000000000000000000000000ecebc6a6b9636667380531fb431aec70c86ab3d7fc883ce8594223459af266a1"
	assert.Equal(t, expected, hex.EncodeToString(priv.PrivateKey))
}

func TestSign(t *testing.T) {

	entropy, err := crypto.RandEntropy(32)
	assert.Equal(t, nil, err)

	priv, err := NewPrivateKey(entropy)
	assert.Equal(t, nil, err)

	sig, err := priv.Sign([]byte("hello world"))
	assert.Equal(t, nil, err)
	assert.Equal(t, 64, len(sig))
}

func TestPrivKeyToAddress(t *testing.T) {
	key := []byte{0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	addr, err := crypto.KeyToAddress(PrivKeyPrefix, key, 1)
	assert.Equal(t, nil, err)
	assert.Equal(t, "DUSKpriv1WCxh37LxDgeLk45Khnyo9cdBG6NC3Ax12ZpGFgD8Mgd7KfapGDg", addr)
}

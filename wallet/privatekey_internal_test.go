package wallet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/crypto"
)

func TestSign(t *testing.T) {
	priv, err := NewPrivateKey()
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

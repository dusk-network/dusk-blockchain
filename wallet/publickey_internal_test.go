package wallet

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/crypto"
)

func TestNewPubKey(t *testing.T) {

	key := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	priv, _ := NewPrivateKey(key)
	expected := "cecc1507dc1ddd7295951c290888f095adb9044d1b73d696e6df065d683bd4fc"
	assert.Equal(t, expected, hex.EncodeToString(priv.PubKey.PublicKey))
}

func TestCorrectVerif(t *testing.T) {

	entropy, err := crypto.RandEntropy(32)
	assert.Equal(t, nil, err)

	priv, _ := NewPrivateKey(entropy)

	msg := []byte("hello world")

	sig, err := priv.Sign(msg)

	res := priv.Public().Verify(msg, sig)

	assert.Equal(t, nil, err)
	assert.Equal(t, true, res)
}
func TestWrongMessage(t *testing.T) {

	entropy, err := crypto.RandEntropy(32)
	assert.Equal(t, nil, err)

	priv, _ := NewPrivateKey(entropy)

	msg := []byte("hello world")

	sig, _ := priv.Sign(msg)

	res := priv.Public().Verify([]byte("hello"), sig)
	assert.Equal(t, false, res)
}
func TestPubKeyToAddress(t *testing.T) {
	key := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	addr, err := crypto.KeyToAddress(PubKeyPrefix, key, 2)
	assert.Equal(t, nil, err)
	assert.Equal(t, "DUSKpub1JBG1FrnwDwtaZnXP3z6NazsXzS3j9B5vBPhszfa3xDpLeCFQkj2M", addr)
}

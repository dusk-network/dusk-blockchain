package wallet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCorrectVerif(t *testing.T) {
	priv, _ := NewPrivateKey()

	msg := []byte("hello world")

	sig, err := priv.Sign(msg)

	res := priv.Public().Verify(msg, sig)

	assert.Equal(t, nil, err)
	assert.Equal(t, true, res)
}
func TestWrongMessage(t *testing.T) {
	priv, _ := NewPrivateKey()

	msg := []byte("hello world")

	sig, _ := priv.Sign(msg)

	res := priv.Public().Verify([]byte("hello"), sig)
	assert.Equal(t, false, res)
}
func TestPubKeyToAddress(t *testing.T) {
	key := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	addr, err := KeyToAddress(PubKeyPrefix, key, 2)
	assert.Equal(t, nil, err)
	assert.Equal(t, "DUSKpub1JBG1FrnwDwtaZnXP3z6NazsXzS3j9B5vBPhszfa3xDpLeCFQkj2M", addr)
}

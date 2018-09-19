package wallet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSign(t *testing.T) {
	priv, err := NewPrivateKey()
	assert.Equal(t, nil, err)

	sig, err := priv.Sign([]byte("hello world"))
	assert.Equal(t, nil, err)
	assert.Equal(t, 64, len(sig))
}

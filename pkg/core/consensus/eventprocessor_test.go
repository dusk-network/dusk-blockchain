package consensus

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

func TestValidator(t *testing.T) {
	message := []byte("This is a test message")
	b := make([]byte, 0)
	tbuf := bytes.NewBuffer(b)
	encoding.WriteVarBytes(tbuf, message)
	testMsg := tbuf.Bytes()

	keys, err := user.NewRandKeys()
	assert.NoError(t, err)
	signature := ed25519.Sign(*keys.EdSecretKey, testMsg)
	assert.Equal(t, 64, len(signature))

	assert.NoError(t, msg.VerifyEd25519Signature(keys.EdPubKeyBytes(), testMsg, signature))

	b = make([]byte, 0)
	buf := bytes.NewBuffer(b)
	assert.NoError(t, encoding.Write512(buf, signature))
	assert.NoError(t, encoding.Write256(buf, keys.EdPubKeyBytes()))
	assert.NoError(t, encoding.WriteVarBytes(buf, message))

	validator := &Validator{}
	result, err := validator.Process(buf)
	assert.NoError(t, err)

	ret := make([]byte, len(message)+1)
	assert.NoError(t, encoding.ReadVarBytes(result, &ret))
	assert.Equal(t, message, ret)
}

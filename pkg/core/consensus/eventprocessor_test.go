package consensus_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

func TestValidator(t *testing.T) {
	message := []byte("This is a test message")

	keys, err := user.NewRandKeys()
	assert.NoError(t, err)
	signature := ed25519.Sign(*keys.EdSecretKey, message)
	assert.Equal(t, 64, len(signature))

	assert.NoError(t, msg.VerifyEd25519Signature(keys.EdPubKeyBytes, message, signature))

	b := make([]byte, 0)
	buf := bytes.NewBuffer(b)
	assert.NoError(t, encoding.Write512(buf, signature))
	assert.NoError(t, encoding.Write256(buf, keys.EdPubKeyBytes))
	_, err = buf.Write(message)
	assert.NoError(t, err)

	validator := &consensus.Validator{}
	result, err := validator.Process(buf)
	assert.NoError(t, err)
	assert.Equal(t, message, result.Bytes())
}

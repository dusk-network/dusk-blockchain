package header

import (
	"bytes"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestSignableVote(t *testing.T) {
	red := &Header{}
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)
	test := &Header{}
	buf := new(bytes.Buffer)
	assert.NoError(t, MarshalSignableVote(buf, red))
	assert.NoError(t, UnmarshalSignableVote(buf, test))
	assert.Equal(t, red, test)
}

func TestUnMarshal(t *testing.T) {
	k, _ := user.NewRandKeys()
	hash, _ := crypto.RandEntropy(32)
	h := &Header{
		PubKeyBLS: k.BLSPubKeyBytes,
		Round:     uint64(1),
		Step:      uint8(2),
		BlockHash: hash,
	}

	buf := new(bytes.Buffer)
	test := &Header{}
	assert.NoError(t, MarshalSignableVote(buf, h))
	assert.NoError(t, UnmarshalSignableVote(buf, test))
	test.PubKeyBLS = k.BLSPubKeyBytes

	assert.Equal(t, h, test)
}

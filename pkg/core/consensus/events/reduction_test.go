package events

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

func TestBlsSignReductionEvent(t *testing.T) {
	k, _ := user.NewRandKeys()
	red := NewReduction()
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)
	red.PubKeyBLS = k.BLSPubKeyBytes

	assert.NoError(t, BlsSignReductionEvent(red, k))
	assert.NotNil(t, red.SignedHash)
	buf := new(bytes.Buffer)
	assert.NoError(t, MarshalSignableVote(buf, red.Header))
	assert.NoError(t, msg.VerifyBLSSignature(red.PubKeyBLS, buf.Bytes(), red.SignedHash))
}

func TestSignReductionEvent(t *testing.T) {
	k, _ := user.NewRandKeys()
	red := NewReduction()
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)
	red.PubKeyBLS = k.BLSPubKeyBytes
	signed, err := SignReductionEvent(red, k)
	assert.NoError(t, err)
	assert.NotNil(t, red.SignedHash)

	sig := make([]byte, 64)
	assert.NoError(t, encoding.Read512(signed, &sig))
	assert.True(t, ed25519.Verify(*k.EdPubKey, signed.Bytes(), sig))

	ev := NewReduction()
	assert.NoError(t, NewReductionUnMarshaller().Unmarshal(signed, ev))
	assert.Equal(t, red, ev)
}

func TestSignReduction(t *testing.T) {
	k, _ := user.NewRandKeys()
	red := NewReduction()
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)

	b := new(bytes.Buffer)
	assert.NoError(t, MarshalSignableVote(b, red.Header))
	assert.NoError(t, SignReduction(b, k))

	// verifying the ED25519 signature
	sig := make([]byte, 64)
	assert.NoError(t, encoding.Read512(b, &sig))
	assert.True(t, ed25519.Verify(*k.EdPubKey, b.Bytes(), sig))

	ev, err := NewReductionUnMarshaller().Deserialize(b)
	assert.NoError(t, err)
	r := ev.(*Reduction)

	//verifying the header
	assert.Equal(t, r.PubKeyBLS, k.BLSPubKeyBytes)
	assert.Equal(t, uint64(1), r.Round)
	assert.Equal(t, uint8(2), r.Step)
	assert.Equal(t, red.BlockHash, r.BlockHash)
}

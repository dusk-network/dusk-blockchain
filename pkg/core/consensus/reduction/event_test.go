package reduction

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/header"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

// This test checks that the UnMarshaller is working properly.
// It tests both the Marshal and Unmarshal method, and compares the events afterwards.
func TestReductionUnMarshal(t *testing.T) {
	unMarshaller := NewUnMarshaller()

	// Mock a Reduction event
	blockHash, err := crypto.RandEntropy(32)
	assert.NoError(t, err)
	h := &header.Header{
		Step:      uint8(4),
		Round:     uint64(120),
		BlockHash: blockHash,
	}
	ev, err := newReductionEvent(h)
	assert.NoError(t, err)

	// Marshal it
	buf := new(bytes.Buffer)
	assert.Nil(t, unMarshaller.Marshal(buf, ev))

	// Now Unmarshal it
	ev2 := New()
	assert.Nil(t, unMarshaller.Unmarshal(buf, ev2))

	// The two events should be the exact same
	assert.Equal(t, ev, ev2)
}

// newReductionEvent returns a Reduction event, populated with a mixture of specified
// and default fields.
func newReductionEvent(h *header.Header) (*Reduction, error) {
	keys, _ := user.NewRandKeys()

	redEv := &Reduction{
		Header: h,
	}

	if err := BlsSign(redEv, keys); err != nil {
		return nil, err
	}

	return redEv, nil
}

func TestBlsSign(t *testing.T) {
	k, _ := user.NewRandKeys()
	red := New()
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)
	red.PubKeyBLS = k.BLSPubKeyBytes

	assert.NoError(t, BlsSign(red, k))
	assert.NotNil(t, red.SignedHash)
	buf := new(bytes.Buffer)
	assert.NoError(t, header.MarshalSignableVote(buf, red.Header))
	assert.NoError(t, msg.VerifyBLSSignature(red.PubKeyBLS, buf.Bytes(), red.SignedHash))
}

func TestSign(t *testing.T) {
	k, _ := user.NewRandKeys()
	red := New()
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)
	red.PubKeyBLS = k.BLSPubKeyBytes
	signed, err := Sign(red, k)
	assert.NoError(t, err)
	assert.NotNil(t, red.SignedHash)

	sig := make([]byte, 64)
	assert.NoError(t, encoding.Read512(signed, &sig))
	assert.True(t, ed25519.Verify(*k.EdPubKey, signed.Bytes(), sig))

	ev := New()
	assert.NoError(t, NewUnMarshaller().Unmarshal(signed, ev))
	assert.Equal(t, red, ev)
}

func TestSignBuffer(t *testing.T) {
	k, _ := user.NewRandKeys()
	red := New()
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)

	b := new(bytes.Buffer)
	assert.NoError(t, header.MarshalSignableVote(b, red.Header))
	assert.NoError(t, SignBuffer(b, k))

	// verifying the ED25519 signature
	sig := make([]byte, 64)
	assert.NoError(t, encoding.Read512(b, &sig))
	assert.True(t, ed25519.Verify(*k.EdPubKey, b.Bytes(), sig))

	ev, err := NewUnMarshaller().Deserialize(b)
	assert.NoError(t, err)
	r := ev.(*Reduction)

	//verifying the header
	assert.Equal(t, r.PubKeyBLS, k.BLSPubKeyBytes)
	assert.Equal(t, uint64(1), r.Round)
	assert.Equal(t, uint8(2), r.Step)
	assert.Equal(t, red.BlockHash, r.BlockHash)
}

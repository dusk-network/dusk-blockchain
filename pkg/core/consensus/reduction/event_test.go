package reduction_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/header"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

// This test checks that the UnMarshaller is working properly.
// It tests both the Marshal and Unmarshal method, and compares the events afterwards.
func TestReductionUnMarshal(t *testing.T) {
	unMarshaller := reduction.NewUnMarshaller()

	// Mock a Reduction event
	ev := newReductionEvent(1, 1)

	// Marshal it
	buf := new(bytes.Buffer)
	assert.NoError(t, unMarshaller.Marshal(buf, ev))

	// Now Unmarshal it
	ev2 := reduction.New()
	assert.NoError(t, unMarshaller.Unmarshal(buf, ev2))

	// The two events should be the exact same
	assert.Equal(t, ev, ev2)
}

// This test ensures proper functionality of marshalling and unmarshalling slices of
// Reduction events.
func TestVoteSetUnMarshal(t *testing.T) {
	unMarshaller := reduction.NewUnMarshaller()

	// Mock a slice of Reduction events
	var evs []wire.Event
	for i := 0; i < 5; i++ {
		ev := newReductionEvent(1, 1)
		evs = append(evs, ev)
	}

	// Marshal it
	buf := new(bytes.Buffer)
	assert.NoError(t, unMarshaller.MarshalVoteSet(buf, evs))

	// Now Unmarshal it
	evs2, err := unMarshaller.UnmarshalVoteSet(buf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, evs, evs2)
}

func TestBlsSign(t *testing.T) {
	k, _ := user.NewRandKeys()
	red := reduction.New()
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)
	red.PubKeyBLS = k.BLSPubKeyBytes

	assert.NoError(t, reduction.BlsSign(red, k))
	assert.NotNil(t, red.SignedHash)
	buf := new(bytes.Buffer)
	assert.NoError(t, header.MarshalSignableVote(buf, red.Header))
	assert.NoError(t, msg.VerifyBLSSignature(red.PubKeyBLS, buf.Bytes(), red.SignedHash))
}

func TestSign(t *testing.T) {
	k, _ := user.NewRandKeys()
	red := reduction.New()
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)
	red.PubKeyBLS = k.BLSPubKeyBytes
	signed, err := reduction.Sign(red, k)
	assert.NoError(t, err)
	assert.NotNil(t, red.SignedHash)

	sig := make([]byte, 64)
	pk := make([]byte, 32)
	assert.NoError(t, encoding.Read512(signed, &sig))
	assert.NoError(t, encoding.Read256(signed, &pk))
	assert.True(t, ed25519.Verify(*k.EdPubKey, signed.Bytes(), sig))

	ev := reduction.New()
	assert.NoError(t, reduction.NewUnMarshaller().Unmarshal(signed, ev))
	assert.Equal(t, red, ev)
}

func TestSignBuffer(t *testing.T) {
	k, _ := user.NewRandKeys()
	red := reduction.New()
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)

	b := new(bytes.Buffer)
	assert.NoError(t, header.MarshalSignableVote(b, red.Header))
	assert.NoError(t, reduction.SignBuffer(b, k))

	// verifying the ED25519 signature
	sig := make([]byte, 64)
	pk := make([]byte, 32)
	assert.NoError(t, encoding.Read512(b, &sig))
	assert.NoError(t, encoding.Read256(b, &pk))
	assert.True(t, ed25519.Verify(*k.EdPubKey, b.Bytes(), sig))

	ev, err := reduction.NewUnMarshaller().Deserialize(b)
	assert.NoError(t, err)
	r := ev.(*reduction.Reduction)

	//verifying the header
	assert.Equal(t, r.PubKeyBLS, k.BLSPubKeyBytes)
	assert.Equal(t, uint64(1), r.Round)
	assert.Equal(t, uint8(2), r.Step)
	assert.Equal(t, red.BlockHash, r.BlockHash)
}

func newReductionEvent(round uint64, step uint8) *reduction.Reduction {
	k, _ := user.NewRandKeys()
	blockHash, _ := crypto.RandEntropy(32)
	return reduction.MockReduction(k, blockHash, round, step)
}

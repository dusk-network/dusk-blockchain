package reduction_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/key"
	"github.com/stretchr/testify/assert"
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
	red, k := newBasicReduction()
	red.PubKeyBLS = k.BLSPubKeyBytes

	assert.NoError(t, reduction.BlsSign(red, k))
	assert.NotNil(t, red.SignedHash)
	buf := new(bytes.Buffer)
	assert.NoError(t, header.MarshalSignableVote(buf, red.Header))
	assert.NoError(t, msg.VerifyBLSSignature(red.PubKeyBLS, buf.Bytes(), red.SignedHash))
}

func TestSign(t *testing.T) {
	red, k := newBasicReduction()
	red.PubKeyBLS = k.BLSPubKeyBytes
	signed, err := reduction.Sign(red, k)
	assert.NoError(t, err)
	assert.NotNil(t, red.SignedHash)

	sig := make([]byte, 64)
	assert.NoError(t, encoding.Read512(signed, sig))
	pk := make([]byte, 32)
	assert.NoError(t, encoding.Read256(signed, pk))
	assert.True(t, ed25519.Verify(*k.EdPubKey, signed.Bytes(), sig))

	ev := reduction.New()
	assert.NoError(t, reduction.NewUnMarshaller().Unmarshal(signed, ev))
	assert.Equal(t, red, ev)
}

func TestSignBuffer(t *testing.T) {
	red, k := newBasicReduction()

	b := new(bytes.Buffer)
	assert.NoError(t, header.MarshalSignableVote(b, red.Header))
	assert.NoError(t, reduction.SignBuffer(b, k))

	// verifying the ED25519 signature
	sig := make([]byte, 64)
	assert.NoError(t, encoding.Read512(b, sig))
	pk := make([]byte, 32)
	assert.NoError(t, encoding.Read256(b, pk))
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
	k, _ := key.NewRandConsensusKeys()
	blockHash, _ := crypto.RandEntropy(32)
	return reduction.MockReduction(k, blockHash, round, step)
}

func newBasicReduction() (*reduction.Reduction, key.ConsensusKeys) {
	k, _ := key.NewRandConsensusKeys()
	red := reduction.New()
	red.Round = uint64(1)
	red.Step = uint8(2)
	red.BlockHash, _ = crypto.RandEntropy(32)
	return red, k
}

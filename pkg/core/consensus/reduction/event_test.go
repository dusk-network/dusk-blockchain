package reduction_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-crypto/bls"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/key"
	"github.com/stretchr/testify/assert"
)

// This test checks that the UnMarshaller is working properly.
// It tests both the Marshal and Unmarshal method, and compares the events afterwards.
func TestReductionUnMarshal(t *testing.T) {
	// Mock a Reduction event
	ev := newReductionEvent(1, 1)

	// Marshal it
	buf := new(bytes.Buffer)
	assert.NoError(t, reduction.Marshal(buf, ev))

	// Now Unmarshal it
	ev2 := reduction.New()
	assert.NoError(t, reduction.Unmarshal(buf, ev2))

	// The two events should be the exact same
	assert.Equal(t, ev, *ev2)
}

// This test ensures proper functionality of marshalling and unmarshalling slices of
// Reduction events.
func TestVoteSetUnMarshal(t *testing.T) {
	// Mock a slice of Reduction events
	var evs []reduction.Reduction
	for i := 0; i < 5; i++ {
		ev := newReductionEvent(1, 1)
		evs = append(evs, ev)
	}

	// Marshal it
	buf := new(bytes.Buffer)
	assert.NoError(t, reduction.MarshalVoteSet(buf, evs))

	// Now Unmarshal it
	evs2, err := reduction.UnmarshalVoteSet(buf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, evs, evs2)
}

func newReductionEvent(round uint64, step uint8) reduction.Reduction {
	k, _ := key.NewRandConsensusKeys()
	blockHash, _ := crypto.RandEntropy(32)
	sig, err := bls.Sign(k.BLSSecretKey, k.BLSPubKey, blockHash)
	if err != nil {
		panic(err)
	}

	return reduction.Reduction{sig.Compress()}
}

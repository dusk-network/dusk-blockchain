package message_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-crypto/bls"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/v2/key"
	"github.com/stretchr/testify/assert"
)

// This test checks that the UnMarshaller is working properly.
// It tests both the Marshal and Unmarshal method, and compares the events afterwards.
func TestReductionUnMarshal(t *testing.T) {
	// Mock a Reduction event
	ev := newReductionEvent(1, 1)

	// Marshal it
	buf := new(bytes.Buffer)
	assert.NoError(t, message.MarshalReduction(buf, ev))

	// Now Unmarshal it
	ev2 := message.NewReduction(header.Header{})
	assert.NoError(t, message.UnmarshalReduction(buf, ev2))

	// The two events should be the exact same
	assert.Equal(t, ev, *ev2)
}

// This test ensures proper functionality of marshalling and unmarshalling slices of
// Reduction events.
func TestVoteSetUnMarshal(t *testing.T) {
	// Mock a slice of Reduction events
	var evs []message.Reduction
	for i := 0; i < 5; i++ {
		ev := newReductionEvent(1, 1)
		evs = append(evs, ev)
	}

	// Marshal it
	buf := new(bytes.Buffer)
	assert.NoError(t, message.MarshalVoteSet(buf, evs))

	// Now Unmarshal it
	evs2, err := message.UnmarshalVoteSet(buf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, evs, evs2)
}

func newReductionEvent(round uint64, step uint8) message.Reduction {
	k, _ := key.NewRandConsensusKeys()
	blockHash, _ := crypto.RandEntropy(32)
	sig, err := bls.Sign(k.BLSSecretKey, k.BLSPubKey, blockHash)
	if err != nil {
		panic(err)
	}
	hdr := header.Header{
		Round:     round,
		Step:      step,
		BlockHash: blockHash,
		PubKeyBLS: k.BLSPubKeyBytes,
	}

	r := message.NewReduction(hdr)
	r.SignedHash = sig.Compress()
	return *r
}

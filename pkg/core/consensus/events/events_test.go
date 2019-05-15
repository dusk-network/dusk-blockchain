package events_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

// This test checks that the ReductionUnMarshaller is working properly.
// It tests both the Marshal and Unmarshal method, and compares the events afterwards.
func TestReductionUnMarshal(t *testing.T) {
	unMarshaller := events.NewReductionUnMarshaller()

	// Mock a Reduction event
	blockHash, err := crypto.RandEntropy(32)
	assert.NoError(t, err)
	h := &events.Header{
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
	ev2 := events.NewReduction()
	assert.Nil(t, unMarshaller.Unmarshal(buf, ev2))

	// The two events should be the exact same
	assert.Equal(t, ev, ev2)
}

// newReductionEvent returns a Reduction event, populated with a mixture of specified
// and default fields.
func newReductionEvent(h *events.Header) (*events.Reduction, error) {
	keys, _ := user.NewRandKeys()

	redEv := &events.Reduction{
		Header: h,
	}

	if err := events.BlsSignReductionEvent(redEv, keys); err != nil {
		return nil, err
	}

	return redEv, nil
}

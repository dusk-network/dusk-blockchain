package events_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// This test checks that the AgreementUnMarshaller is working properly.
// It tests both the Marshal and Unmarshal method, and compares the events afterwards.
func TestAgreementUnMarshal(t *testing.T) {
	unMarshaller := events.NewAgreementUnMarshaller()

	// Mock an Agreement event
	blockHash, err := crypto.RandEntropy(32)
	assert.Nil(t, err)
	ev, err := newAgreementEvent(blockHash, 120, 4)
	assert.Nil(t, err)

	// Marshal it
	buf := new(bytes.Buffer)
	assert.Nil(t, unMarshaller.Marshal(buf, ev))

	// Now Unmarshal it
	ev2 := events.NewAgreement()
	assert.Nil(t, unMarshaller.Unmarshal(buf, ev2))

	// The two events should be the exact same
	assert.Equal(t, ev, ev2)
}

// This test checks that the ReductionUnMarshaller is working properly.
// It tests both the Marshal and Unmarshal method, and compares the events afterwards.
func TestReductionUnMarshal(t *testing.T) {
	unMarshaller := events.NewReductionUnMarshaller()

	// Mock a Reduction event
	blockHash, err := crypto.RandEntropy(32)
	assert.Nil(t, err)
	ev, err := newReductionEvent(blockHash, 120, 4)
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

// newAgreementEvent returns an Agreement event, populated with the specified fields.
// The event can be used for marshallers to test their functionality.
func newAgreementEvent(blockHash []byte, round uint64, step uint8) (*events.Agreement, error) {
	pubKeyBLS, byte33, err := generateEventFields()
	if err != nil {
		return nil, err
	}

	vote, err := newReductionEvent(blockHash, round, step)
	if err != nil {
		return nil, err
	}

	return &events.Agreement{
		Header: &events.Header{
			Round:     round,
			Step:      step,
			PubKeyBLS: pubKeyBLS,
		},
		SignedVoteSet: byte33,
		VoteSet:       []wire.Event{vote},
		AgreedHash:    blockHash,
	}, nil
}

// newReductionEvent returns a Reduction event, populated with a mixture of specified
// and default fields.
func newReductionEvent(hash []byte, round uint64, step uint8) (*events.Reduction, error) {
	pubKeyBLS, byte33, err := generateEventFields()
	if err != nil {
		return nil, err
	}

	return &events.Reduction{
		Header: &events.Header{
			PubKeyBLS: pubKeyBLS,
			Round:     round,
			Step:      step,
		},
		VotedHash:  hash,
		SignedHash: byte33,
	}, nil
}

// generateEventFields will create byte slices needed to mock Events.
func generateEventFields() ([]byte, []byte, error) {
	pub, _, err := bls.GenKeyPair(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	byte33, err := crypto.RandEntropy(33)
	if err != nil {
		return nil, nil, err
	}

	return pub.Marshal(), byte33, nil
}

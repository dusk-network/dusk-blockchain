package reduction

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// reductionEventCollector is a helper for common operations on stored Event Arrays
type reductionEventCollector map[string]wire.Event

// Clear up the Collector
func (rec reductionEventCollector) Clear() {
	for key := range rec {
		delete(rec, key)
	}
}

// Contains checks if we already collected an event for this public key
func (rec reductionEventCollector) Contains(pubKey string) bool {
	if rec[pubKey] != nil {
		return true
	}

	return false
}

// Store the Event keeping track of the step it belongs to. It silently ignores duplicates (meaning it does not store an event in case it is already found at the step specified). It returns the number of events stored at specified step *after* the store operation
func (rec reductionEventCollector) Store(event wire.Event, pubKey string) {
	if rec.Contains(pubKey) {
		return
	}

	rec[pubKey] = event
}

// Event is a basic reduction event.
type Event struct {
	*consensus.EventHeader
	VotedHash  []byte
	SignedHash []byte
}

// Equal as specified in the Event interface
func (e *Event) Equal(ev wire.Event) bool {
	other, ok := ev.(*Event)
	return ok && (bytes.Equal(e.PubKeyBLS, other.PubKeyBLS)) && (e.Round == other.Round) && (e.Step == other.Step)
}

type reductionEventUnmarshaller struct {
	*consensus.EventHeaderUnmarshaller
}

func newReductionEventUnmarshaller(validate func(*bytes.Buffer) error) *reductionEventUnmarshaller {
	return &reductionEventUnmarshaller{
		EventHeaderUnmarshaller: consensus.NewEventHeaderUnmarshaller(validate),
	}
}

// Unmarshal unmarshals the buffer into a CommitteeEvent
func (a *reductionEventUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	rev := ev.(*Event)
	if err := a.EventHeaderUnmarshaller.Unmarshal(r, rev.EventHeader); err != nil {
		return err
	}

	if err := encoding.Read256(r, &rev.VotedHash); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &rev.SignedHash); err != nil {
		return err
	}

	return nil
}

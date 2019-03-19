package reduction

import (
	"bytes"
	"encoding/binary"
	"errors"

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

type Event struct {
	VotedHash  []byte
	SignedHash []byte
	PubKeyBLS  []byte
	Round      uint64
	Step       uint8
}

// Equal as specified in the Event interface
func (e *Event) Equal(ev wire.Event) bool {
	other, ok := ev.(*Event)
	return ok && (bytes.Equal(e.PubKeyBLS, other.PubKeyBLS)) && (e.Round == other.Round) && (e.Step == other.Step)
}

type reductionEventUnmarshaller struct {
	validate func(*bytes.Buffer) error
}

func newReductionEventUnmarshaller(validate func(*bytes.Buffer) error) *reductionEventUnmarshaller {
	return &reductionEventUnmarshaller{validate}
}

// Unmarshal unmarshals the buffer into a CommitteeEvent
func (a *reductionEventUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	if err := a.validate(r); err != nil {
		return err
	}

	reductionEvent, ok := ev.(*Event)
	if !ok {
		return errors.New("type casting event failed")
	}

	if err := encoding.Read256(r, &reductionEvent.VotedHash); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &reductionEvent.SignedHash); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &reductionEvent.PubKeyBLS); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &reductionEvent.Round); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &reductionEvent.Step); err != nil {
		return err
	}

	return nil
}

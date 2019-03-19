package reduction

import (
	"bytes"
	"encoding/binary"
	"errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// reductionEventCollector is an helper for common operations on stored Event Arrays
type reductionEventCollector map[string][]wire.Event

// Clear up the Collector
func (rec reductionEventCollector) Clear() {
	for key := range rec {
		delete(rec, key)
	}
}

// Contains checks if we already collected this event
func (rec reductionEventCollector) Contains(event wire.Event, hash string) bool {
	for _, stored := range rec[hash] {
		if event.Equal(stored) {
			return true
		}
	}

	return false
}

// Store the Event keeping track of the step it belongs to. It silently ignores duplicates (meaning it does not store an event in case it is already found at the step specified). It returns the number of events stored at specified step *after* the store operation
func (rec reductionEventCollector) Store(event wire.Event, hash string) int {
	eventList := rec[hash]
	if rec.Contains(event, hash) {
		return len(eventList)
	}

	if eventList == nil {
		eventList = make([]wire.Event, 0, 100)
	}

	// storing the agreement vote for the proper step
	eventList = append(eventList, event)
	rec[hash] = eventList
	return len(eventList)
}

type reductionEvent struct {
	VotedHash  []byte
	SignedHash []byte
	PubKeyBLS  []byte
	Round      uint64
	Step       uint8
}

// Equal as specified in the Event interface
func (r *reductionEvent) Equal(e wire.Event) bool {
	other, ok := e.(*reductionEvent)
	return ok && (bytes.Equal(r.PubKeyBLS, other.PubKeyBLS)) && (r.Round == other.Round) && (r.Step == other.Step)
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

	reductionEvent, ok := ev.(*reductionEvent)
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

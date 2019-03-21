package reduction

import (
	"bytes"
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// reductionEventCollector is a helper for common operations on stored Event Arrays
	reductionEventCollector map[string]wire.Event

	// Event is a basic reduction event.
	Event struct {
		*consensus.EventHeader
		VotedHash  []byte
		SignedHash []byte
	}

	reductionEventUnmarshaller struct {
		*consensus.EventHeaderUnmarshaller
	}

	// SigSetReduction is a reduction event for the signature set phase.
	SigSetReduction struct {
		*Event
		winningBlockHash []byte
	}

	sigSetReductionUnmarshaller struct {
		*reductionEventUnmarshaller
	}
)

// Clear up the Collector
func (rec reductionEventCollector) Clear() {
	for key := range rec {
		delete(rec, key)
	}
}

// Contains checks if we already collected an event for this public key
func (rec reductionEventCollector) Contains(pubKey []byte) bool {
	pubKeyStr := hex.EncodeToString(pubKey)
	if rec[pubKeyStr] != nil {
		return true
	}

	return false
}

// Store the Event keeping track of the step it belongs to. It silently ignores duplicates (meaning it does not store an event in case it is already found at the step specified). It returns the number of events stored at specified step *after* the store operation
func (rec reductionEventCollector) Store(event wire.Event, pubKey []byte) {
	if rec.Contains(pubKey) {
		return
	}

	pubKeyStr := hex.EncodeToString(pubKey)
	rec[pubKeyStr] = event
}

// Equal as specified in the Event interface
func (e *Event) Equal(ev wire.Event) bool {
	other, ok := ev.(*Event)
	return ok && (bytes.Equal(e.PubKeyBLS, other.PubKeyBLS)) && (e.Round == other.Round) && (e.Step == other.Step)
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

// Equal implements Event interface.
func (ssr *SigSetReduction) Equal(e wire.Event) bool {
	return ssr.Event.Equal(e) &&
		bytes.Equal(ssr.winningBlockHash, e.(*SigSetReduction).winningBlockHash)
}

func newSigSetReductionUnmarshaller(validate func(*bytes.Buffer) error) *reductionEventUnmarshaller {
	return newReductionEventUnmarshaller(validate)
}

func (ssru *sigSetReductionUnmarshaller) Unmarshal(r *bytes.Buffer, e wire.Event) error {
	sigSetReduction := e.(*SigSetReduction)

	if err := ssru.reductionEventUnmarshaller.Unmarshal(r, sigSetReduction.Event); err != nil {
		return err
	}

	if err := encoding.Read256(r, &sigSetReduction.winningBlockHash); err != nil {
		return err
	}

	return nil
}

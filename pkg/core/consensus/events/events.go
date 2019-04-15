package events

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	// Header is an embeddable struct representing the consensus event header fields
	Header struct {
		PubKeyBLS []byte
		Round     uint64
		Step      uint8
	}

	// Agreement is the message that encapsulates data relevant for
	// components relying on committee information
	Agreement struct {
		*Header
		VoteSet       []wire.Event
		SignedVoteSet []byte
		AgreedHash    []byte
	}

	// Reduction is a basic reduction event.
	Reduction struct {
		*Header
		VotedHash  []byte
		SignedHash []byte
	}
)

// NewReduction returns and empty Reduction event.
func NewReduction() *Reduction {
	return &Reduction{
		Header: &Header{},
	}
}

// NewAgreement returns an empty Agreement event.
func NewAgreement() *Agreement {
	return &Agreement{
		Header: &Header{},
	}
}

// Equal as specified in the Event interface
func (a *Header) Equal(e wire.Event) bool {
	other, ok := e.(*Header)
	return ok && (bytes.Equal(a.PubKeyBLS, other.PubKeyBLS)) &&
		(a.Round == other.Round) && (a.Step == other.Step)
}

// Sender of the Event
func (a *Header) Sender() []byte {
	return a.PubKeyBLS
}

// Equal as specified in the Event interface
func (e *Reduction) Equal(ev wire.Event) bool {
	other, ok := ev.(*Reduction)
	return ok && (bytes.Equal(e.PubKeyBLS, other.PubKeyBLS)) &&
		(e.Round == other.Round) && (e.Step == other.Step)
}

// Equal as specified in the Event interface
func (ceh *Agreement) Equal(e wire.Event) bool {
	other, ok := e.(*Agreement)
	return ok && ceh.Header.Equal(other) &&
		bytes.Equal(other.SignedVoteSet, ceh.SignedVoteSet)
}

package selection

import (
	"bytes"
	"errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	// SigSetHandler is the
	SigSetHandler struct {
		committee committee.Committee
		blockHash []byte
	}

	// SigSetEvent expresses a vote on a block hash. It is a real type alias of notaryEvent
	SigSetEvent = committee.Event

	// SigSetUnMarshaller is the unmarshaller of BlockEvents. It is a real type alias of notaryEventUnmarshaller
	SigSetUnMarshaller = committee.EventUnMarshaller
)

// NewSigSetHandler creates a new SigSetHandler and wires it up to phase updates
func NewSigSetHandler(c committee.Committee, eventBus *wire.EventBus) *SigSetHandler {
	phaseChan := consensus.InitPhaseCollector(eventBus).BlockHashChan
	sigSetHandler := &SigSetHandler{committee: c, blockHash: nil}
	go func() {
		sigSetHandler.blockHash = <-phaseChan
	}()
	return sigSetHandler
}

// NewEvent creates a new SigSetEvent struct prototype
func (s *SigSetHandler) NewEvent() wire.Event {
	return &ScoreEvent{}
}

// Stage extracts the Round and Step information from an Event
func (s *SigSetHandler) Stage(e wire.Event) (uint64, uint8) {
	sse := e.(*SigSetEvent)
	return sse.Round, sse.Step
}

// Priority is used to prioritize events. In the case of SigSetHandler, the priority is delegated to the Committee considering that they decide which vote selection should be chosen
func (s *SigSetHandler) Priority(first wire.Event, second wire.Event) wire.Event {
	return s.committee.Priority(first, second)
}

// Verify performs a type conversion to SigSetEvent and delegates the Committee to verify the vote set (plus checks the relevance of a event in respect to the current round)
func (s *SigSetHandler) Verify(event wire.Event) error {
	// validating if the event is related to the current winning block hash
	ev, ok := event.(*SigSetEvent)
	if !ok {
		return errors.New("Mismatched Type: expected SigSetEvent")
	}
	if !bytes.Equal(s.blockHash, ev.BlockHash) {
		return errors.New("Vote set is for the wrong block hash")
	}

	// delegating the committee to verify the vote set
	if err := s.committee.VerifyVoteSet(ev.VoteSet, ev.SignedVoteSet, ev.Round, ev.Step); err != nil {
		return err
	}

	if len(ev.VoteSet) < s.committee.Quorum() {
		// TODO: should we serialize the event into a string?
		return errors.New("Signature selection: vote set is too small")
	}
	return nil
}

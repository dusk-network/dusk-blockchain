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

func NewSigSetHandler(c committee.Committee, eventBus *wire.EventBus) *SigSetHandler {
	phaseChan := consensus.InitPhaseCollector(eventBus).BlockHashChan
	sigSetHandler := &SigSetHandler{committee: c, blockHash: nil}
	go func() {
		sigSetHandler.blockHash = <-phaseChan
	}()
	return sigSetHandler
}

func (s *SigSetHandler) NewEvent() wire.Event {
	return &ScoreEvent{}
}

func (s *SigSetHandler) Stage(e wire.Event) (uint64, uint8) {
	sse := e.(*SigSetEvent)
	return sse.Round, sse.Step
}

func (s *SigSetHandler) Priority(first wire.Event, second wire.Event) wire.Event {
	return s.committee.Priority(first, second)
}

func (s *SigSetHandler) Verify(event wire.Event, hash []byte) error {
	// validating if the event is related to the current winning block hash
	ev, ok := event.(*SigSetEvent)
	if !ok {
		return errors.New("Mismatched Type: expected SigSetEvent")
	}
	if !bytes.Equal(s.blockHash, ev.BlockHash) {
		return errors.New("vote set is for the wrong block hash")
	}

	// delegating the committee to verify the vote set
	if err := s.committee.VerifyVoteSet(ev.VoteSet, ev.SignedVoteSet, ev.Round, ev.Step); err != nil {
		return err
	}

	if len(ev.VoteSet) < s.committee.Quorum() {
		// TODO: should we serialize the event into a string?
		return errors.New("signature selection: vote set is too small")
	}
	return nil
}

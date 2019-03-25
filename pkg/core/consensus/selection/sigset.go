package selection

import (
	"bytes"
	"errors"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// LaunchSignatureSelector creates the component which responsibility is to collect the signatures until a quorum is reached
func LaunchSignatureSelector(c committee.Committee, eventBus *wire.EventBus, timeout time.Duration) *broker {
	handler := newSigSetHandler(c, eventBus)
	broker := newBroker(eventBus, handler, timeout, string(msg.SigSetSelectionTopic))
	go broker.Listen()
	return broker
}

//InitSigSetSelection is a utility function to create and wire up a SigSetEvent channel ready to yield the best SigSetEvent
func InitBestSigSetUpdate(eventBus *wire.EventBus) chan []byte {
	bestVotedHashChan := make(chan []byte, 1)
	selectionCollector := &sigSetCollector{
		bestVotedHashChan: bestVotedHashChan,
	}
	go wire.NewEventSubscriber(eventBus, selectionCollector,
		string(msg.SigSetSelectionTopic)).Accept()
	return bestVotedHashChan
}

type (
	// SigSetHandler is the
	SigSetHandler struct {
		committee    committee.Committee
		blockHash    []byte
		unMarshaller *SigSetUnMarshaller
	}

	// SigSetEvent expresses a vote on a block hash. It is a real type alias of notaryEvent.
	// TODO: SigSetEvent is different than a Committee Event although the outline is the same. In fact the SignedVoteSet is absent from the selection and in its place there should be the VoteHash
	SigSetEvent = committee.Event

	// SigSetUnMarshaller is the unmarshaller of BlockEvents. It is a real type alias of notaryEventUnmarshaller
	SigSetUnMarshaller = committee.EventUnMarshaller

	// sigSetCollector is the private struct helping the plumbing of the SigSet channel whereto public selected SigSetEvent get published
	sigSetCollector struct {
		bestVotedHashChan chan []byte
	}
)

// newSigSetHandler creates a new SigSetHandler and wires it up to phase updates
func newSigSetHandler(c committee.Committee, eventBus *wire.EventBus) *SigSetHandler {
	phaseChan := consensus.InitPhaseUpdate(eventBus)
	sigSetHandler := &SigSetHandler{
		committee: c,
		blockHash: nil,
		// TODO: get rid of validateFunc
		unMarshaller: committee.NewEventUnMarshaller(),
	}
	go func() {
		sigSetHandler.blockHash = <-phaseChan
	}()
	return sigSetHandler
}

// Unmarshal a buffer into a SigSetEvent
func (s *SigSetHandler) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	return s.unMarshaller.Unmarshal(r, ev)
}

// Marshal a SigSetEvent into a Buffer
func (s *SigSetHandler) Marshal(r *bytes.Buffer, ev wire.Event) error {
	return s.unMarshaller.Marshal(r, ev)
}

// NewEvent creates a new SigSetEvent struct prototype
func (s *SigSetHandler) NewEvent() wire.Event {
	return &SigSetEvent{}
}

// ExtractHeader extracts the Round and Step information from an Event
func (s *SigSetHandler) ExtractHeader(e wire.Event, h *consensus.EventHeader) {
	ev := e.(*SigSetEvent)
	h.Round = ev.Round
	h.Step = ev.Step
	h.PubKeyBLS = ev.PubKeyBLS
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

// Collect a message and transform it into a selection message to be consumed by the other components.
func (ssc *sigSetCollector) Collect(r *bytes.Buffer) error {
	ev := &SigSetEvent{}
	unmarshaller := committee.NewEventUnMarshaller()
	if err := unmarshaller.Unmarshal(r, ev); err != nil {
		return err
	}
	// SignedVoteSet is actually the votedHash
	ssc.bestVotedHashChan <- ev.SignedVoteSet
	return nil
}

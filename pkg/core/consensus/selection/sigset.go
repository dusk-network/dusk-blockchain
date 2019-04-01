package selection

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// LaunchSignatureSelector creates the component which responsibility is to collect the signatures until a quorum is reached
func LaunchSignatureSelector(c committee.Committee, eventBus *wire.EventBus, timeout time.Duration) *sigSetBroker {
	handler := newSigSetHandler(c, eventBus)
	broker := newSigSetBroker(eventBus, handler, timeout)
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

func InitSigSetGenerationCollector(eventBus *wire.EventBus) chan bool {
	selectionChan := make(chan bool, 1)
	collector := &selectionCollector{selectionChan}
	go wire.NewEventSubscriber(eventBus, collector, msg.SigSetGenerationTopic).Accept()
	return selectionChan
}

// newSigSetBroker creates a Broker component which responsibility is to listen to the eventbus and supervise Collector operations
func newSigSetBroker(eventBus *wire.EventBus, handler consensus.EventHandler, timeout time.Duration) *sigSetBroker {
	//creating the channel whereto notifications about round updates are push onto
	roundChan := consensus.InitRoundUpdate(eventBus)
	phaseChan := consensus.InitPhaseUpdate(eventBus)
	generationChan := InitSigSetGenerationCollector(eventBus)
	collector := initCollector(handler, timeout, eventBus, string(topics.SigSet))

	return &sigSetBroker{
		eventBus:        eventBus,
		collector:       collector,
		roundUpdateChan: roundChan,
		phaseUpdateChan: phaseChan,
		generationChan:  generationChan,
	}
}

// Listen on the eventBus for relevant topics to feed the collector
func (f *sigSetBroker) Listen() {
	for {
		select {
		case roundUpdate := <-f.roundUpdateChan:
			f.collector.UpdateRound(roundUpdate)
		case phaseUpdate := <-f.phaseUpdateChan:
			f.collector.CurrentBlockHash = phaseUpdate
			f.collector.StartSelection()
		case <-f.generationChan:
			f.collector.StartSelection()
		case bestEvent := <-f.collector.BestEventChan:
			// TODO: remove
			fmt.Println("selected set")

			// TODO: moved step incrementation here, so we dont run ahead in case
			// there's nobody generating blocks
			if bestEvent.Len() != 0 {
				f.collector.CurrentStep++
				f.eventBus.Publish(msg.SigSetSelectionTopic, bestEvent)
			} else {
				f.eventBus.Publish(msg.SigSetGenerationTopic, nil)
			}
		}
	}
}

type (
	// SigSetHandler aggregates operations specific to the SigSet Selection operations
	SigSetHandler struct {
		committee    committee.Committee
		blockHash    []byte
		unMarshaller *SigSetUnMarshaller
	}

	// SigSetEvent expresses a vote on a block hash. It is a real type alias of notaryEvent.
	SigSetEvent = committee.NotaryEvent

	// SigSetUnMarshaller is the unmarshaller of BlockEvents. It is a real type alias of notaryEventUnmarshaller
	SigSetUnMarshaller = committee.NotaryEventUnMarshaller

	// sigSetCollector is the private struct helping the plumbing of the SigSet channel whereto public selected SigSetEvent get published
	sigSetCollector struct {
		bestVotedHashChan chan []byte
	}

	// broker is the component that supervises a collection of events
	sigSetBroker struct {
		eventBus        *wire.EventBus
		phaseUpdateChan <-chan []byte
		roundUpdateChan <-chan uint64
		generationChan  <-chan bool
		collector       *collector
	}
)

// newSigSetHandler creates a new SigSetHandler and wires it up to phase updates
func newSigSetHandler(c committee.Committee, eventBus *wire.EventBus) *SigSetHandler {
	phaseChan := consensus.InitPhaseUpdate(eventBus)
	sigSetHandler := &SigSetHandler{
		committee: c,
		blockHash: nil,
		// TODO: get rid of validateFunc
		unMarshaller: committee.NewNotaryEventUnMarshaller(msg.VerifyEd25519Signature),
	}
	go func() {
		for {
			sigSetHandler.blockHash = <-phaseChan
		}
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
	return &SigSetEvent{
		EventHeader: &consensus.EventHeader{},
	}
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
	ev := &SigSetEvent{
		EventHeader: &consensus.EventHeader{},
	}
	unmarshaller := committee.NewNotaryEventUnMarshaller(msg.VerifyEd25519Signature)
	if err := unmarshaller.Unmarshal(r, ev); err != nil {
		return err
	}
	// SignedVoteSet is actually the votedHash
	ssc.bestVotedHashChan <- ev.SignedVoteSet
	return nil
}

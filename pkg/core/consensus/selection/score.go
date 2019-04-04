package selection

import (
	"bytes"
	"errors"
	"math/big"
	"time"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
)

type (
	scoreHandler struct {
		bidList      user.BidList
		unMarshaller *ScoreUnMarshaller
	}

	bidListCollector struct {
		BidListChan chan user.BidList
	}

	//scoreCollector is a helper to obtain a score channel already wired to the EventBus and fully functioning
	scoreCollector struct {
		bestVotedScoreHashChan chan []byte
	}

	scoreSelectionCollector struct {
		scoreSelectionChan chan bool
	}

	// broker is the component that supervises a collection of events
	scoreBroker struct {
		eventBus        *wire.EventBus
		roundUpdateChan <-chan uint64
		generationChan  <-chan bool
		collector       *collector
	}
)

// LaunchScoreSelectionComponent creates and launches the component which responsibility is to validate and select the best score among the blind bidders. The component publishes under the topic BestScoreTopic
func LaunchScoreSelectionComponent(eventBus *wire.EventBus, timeout time.Duration, bidList user.BidList) *scoreBroker {
	handler := newScoreHandler(eventBus, bidList)
	broker := newScoreBroker(eventBus, handler, timeout)
	go broker.Listen()
	return broker
}

// InitBestScoreUpdate is the utility function to create and wire a channel for notifications of the best ScoreEvent
func InitBestScoreUpdate(eventBus *wire.EventBus) chan []byte {
	bestVotedScoreHashChan := make(chan []byte, 1)
	collector := &scoreCollector{
		bestVotedScoreHashChan: bestVotedScoreHashChan,
	}
	go wire.NewEventSubscriber(eventBus, collector, string(msg.BestScoreTopic)).Accept()
	return bestVotedScoreHashChan
}

// InitBidListUpdate creates and initiates a channel for the updates in the BidList
func InitBidListUpdate(eventBus *wire.EventBus) chan user.BidList {
	bidListChan := make(chan user.BidList)
	collector := &bidListCollector{bidListChan}
	go wire.NewEventSubscriber(eventBus, collector, string(msg.BidListTopic)).Accept()
	return bidListChan
}

func InitBlockGenerationCollector(eventBus *wire.EventBus) chan bool {
	selectionChan := make(chan bool, 1)
	collector := &selectionCollector{selectionChan}
	go wire.NewEventSubscriber(eventBus, collector, msg.BlockGenerationTopic).Accept()
	return selectionChan
}

// newScoreBroker creates a Broker component which responsibility is to listen to the eventbus and supervise Collector operations
func newScoreBroker(eventBus *wire.EventBus, handler consensus.EventHandler, timeout time.Duration) *scoreBroker {
	//creating the channel whereto notifications about round updates are push onto
	roundChan := consensus.InitRoundUpdate(eventBus)
	generationChan := InitBlockGenerationCollector(eventBus)
	collector := initCollector(handler, timeout, eventBus, string(topics.Score))

	return &scoreBroker{
		eventBus:        eventBus,
		collector:       collector,
		roundUpdateChan: roundChan,
		generationChan:  generationChan,
	}
}

// Listen on the eventBus for relevant topics to feed the collector
func (f *scoreBroker) Listen() {
	for {
		select {
		case roundUpdate := <-f.roundUpdateChan:
			f.collector.UpdateRound(roundUpdate)
			f.collector.StartSelection()
		case <-f.generationChan:
			f.collector.StartSelection()
		case bestEvent := <-f.collector.BestEventChan:
			// TODO: moved step incrementation here, so we dont run ahead in case there's nobody generating blocks
			if bestEvent.Len() != 0 {
				f.collector.CurrentStep++
				f.eventBus.Publish(msg.BestScoreTopic, bestEvent)
			} else {
				f.eventBus.Publish(msg.BlockGenerationTopic, nil)
			}
		case ev := <-f.collector.RepropagateChan:
			// TODO: review
			buf := new(bytes.Buffer)
			if err := f.collector.handler.Marshal(buf, ev); err != nil {
				panic(err)
			}
			message, _ := wire.AddTopic(buf, topics.Score)
			f.eventBus.Publish(string(topics.Gossip), message)
		}
	}
}

func (sc *scoreCollector) Collect(r *bytes.Buffer) error {
	ev := &ScoreEvent{}
	unmarshaller := newScoreUnMarshaller()
	if err := unmarshaller.Unmarshal(r, ev); err != nil {
		return err
	}
	sc.bestVotedScoreHashChan <- ev.VoteHash
	return nil
}

// Collect as defined in the EventCollector interface. It reconstructs the bidList and notifies about it
func (l *bidListCollector) Collect(r *bytes.Buffer) error {
	bidList, err := user.ReconstructBidListSubset(r.Bytes())
	if err != nil {
		return nil
	}
	l.BidListChan <- bidList
	return nil
}

// NewScoreHandler returns a ScoreHandler, which encapsulates specific operations (e.g. verification, validation, marshalling and unmarshalling)
func newScoreHandler(eventBus *wire.EventBus, bidList user.BidList) *scoreHandler {
	bidListChan := InitBidListUpdate(eventBus)
	sh := &scoreHandler{
		unMarshaller: newScoreUnMarshaller(),
		bidList:      bidList,
	}
	go func() {
		for {
			sh.bidList = <-bidListChan
		}
	}()
	return sh
}

func (p *scoreHandler) NewEvent() wire.Event {
	return &ScoreEvent{}
}

func (p *scoreHandler) Unmarshal(r *bytes.Buffer, e wire.Event) error {
	return p.unMarshaller.Unmarshal(r, e)
}

func (p *scoreHandler) Marshal(r *bytes.Buffer, e wire.Event) error {
	return p.unMarshaller.Marshal(r, e)
}

func (p *scoreHandler) ExtractHeader(e wire.Event, h *consensus.EventHeader) {
	ev := e.(*ScoreEvent)
	h.Round = ev.Round
	h.Step = ev.Step
	h.PubKeyBLS = ev.Z
}

// Priority returns true if the
func (p *scoreHandler) Priority(first, second wire.Event) wire.Event {
	ev1, ok := first.(*ScoreEvent)
	if !ok {
		// this happens when first is nil, in which case we should return second
		return second
	}

	ev2 := second.(*ScoreEvent)
	score1 := big.NewInt(0).SetBytes(ev1.Score).Uint64()
	score2 := big.NewInt(0).SetBytes(ev2.Score).Uint64()
	if score1 < score2 {
		return ev2
	}

	return ev1
}

func (p *scoreHandler) Verify(ev wire.Event) error {
	m := ev.(*ScoreEvent)

	// Check first if the BidList contains valid bids
	if err := p.validateBidListSubset(m.BidListSubset); err != nil {
		return err
	}

	// Verify the proof
	seedScalar := ristretto.Scalar{}
	seedScalar.Derive(m.Seed)
	if !zkproof.Verify(m.Proof, seedScalar.Bytes(), m.BidListSubset, m.Score, m.Z) {
		return errors.New("proof verification failed")
	}

	return nil
}

func (p *scoreHandler) validateBidListSubset(bidListSubsetBytes []byte) *prerror.PrError {
	bidListSubset, err := user.ReconstructBidListSubset(bidListSubsetBytes)
	if err != nil {
		return err
	}

	return p.bidList.ValidateBids(bidListSubset)
}

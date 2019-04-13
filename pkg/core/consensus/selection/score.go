package selection

import (
	"bytes"
	"errors"
	"math/big"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
	"gitlab.dusk.network/dusk-core/zkproof"
)

type (
	scoreHandler struct {
		sync.RWMutex
		bidList      user.BidList
		unMarshaller *ScoreUnMarshaller
	}

	bidListCollector struct {
		BidListChan chan<- user.BidList
	}

	//scoreCollector is a helper to obtain a score channel already wired to the EventBus and fully functioning
	scoreCollector struct {
		bestVotedScoreHashChan chan<- []byte
	}

	// broker is the component that supervises a collection of events
	scoreBroker struct {
		roundUpdateChan <-chan uint64
		generationChan  <-chan state
		bidListChan     <-chan user.BidList
		collector       *collector
	}
)

// LaunchScoreSelectionComponent creates and launches the component which responsibility is to validate and select the best score among the blind bidders. The component publishes under the topic BestScoreTopic
func LaunchScoreSelectionComponent(eventBroker wire.EventBroker, timeout time.Duration) *scoreBroker {
	handler := newScoreHandler()
	broker := newScoreBroker(eventBroker, handler, timeout)
	go broker.Listen()
	return broker
}

// InitBestScoreUpdate is the utility function to create and wire a channel for notifications of the best ScoreEvent
func InitBestScoreUpdate(subscriber wire.EventSubscriber) chan []byte {
	bestVotedScoreHashChan := make(chan []byte, 1)
	collector := &scoreCollector{
		bestVotedScoreHashChan: bestVotedScoreHashChan,
	}
	go wire.NewTopicListener(subscriber, collector, string(msg.BestScoreTopic)).Accept()
	return bestVotedScoreHashChan
}

// InitBidListUpdate creates and initiates a channel for the updates in the BidList
func InitBidListUpdate(subscriber wire.EventSubscriber) chan user.BidList {
	bidListChan := make(chan user.BidList)
	collector := &bidListCollector{bidListChan}
	go wire.NewTopicListener(subscriber, collector, string(msg.BidListTopic)).Accept()
	return bidListChan
}

func InitBlockGenerationCollector(subscriber wire.EventSubscriber) chan state {
	selectionChan := make(chan state, 1)
	collector := &selectionCollector{selectionChan}
	go wire.NewTopicListener(subscriber, collector, msg.BlockGenerationTopic).Accept()
	return selectionChan
}

// newScoreBroker creates a Broker component which responsibility is to listen to the eventbus and supervise Collector operations
func newScoreBroker(eventBroker wire.EventBroker, handler scoreEventHandler, timeout time.Duration) *scoreBroker {
	//creating the channel whereto notifications about round updates are push onto
	roundChan := consensus.InitRoundUpdate(eventBroker)
	generationChan := InitBlockGenerationCollector(eventBroker)
	bidListChan := InitBidListUpdate(eventBroker)
	collector := initCollector(handler, timeout, eventBroker, string(topics.Score), consensus.NewState())

	return &scoreBroker{
		collector:       collector,
		roundUpdateChan: roundChan,
		bidListChan:     bidListChan,
		generationChan:  generationChan,
	}
}

// Listen on the eventBus for relevant topics to feed the collector
func (f *scoreBroker) Listen() {
	for {
		select {
		case roundUpdate := <-f.roundUpdateChan:
			f.collector.UpdateRound(roundUpdate)
		case state := <-f.generationChan:
			log.WithFields(log.Fields{
				"process": "selection",
				"round":   state.round,
				"step":    state.step,
			}).Debugln("received regeneration message")
			f.collector.selector.RLock()
			if !f.collector.selector.running {
				go f.collector.selector.startSelection()
			}
			f.collector.selector.RUnlock()
		case bidList := <-f.bidListChan:
			f.collector.selector.handler.UpdateBidList(bidList)
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
func newScoreHandler() *scoreHandler {
	return &scoreHandler{
		RWMutex:      sync.RWMutex{},
		unMarshaller: newScoreUnMarshaller(),
	}
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

func (p *scoreHandler) UpdateBidList(bidList user.BidList) {
	p.Lock()
	defer p.Unlock()
	p.bidList = bidList
}

func (p *scoreHandler) ExtractHeader(e wire.Event, h *consensus.EventHeader) {
	ev := e.(ScoreEvent)
	h.Round = ev.Round
}

// Priority returns true if the
func (p *scoreHandler) Priority(first, second wire.Event) wire.Event {
	ev1, ok := first.(ScoreEvent)
	if !ok {
		// this happens when first is nil, in which case we should return second
		return second
	}

	ev2 := second.(ScoreEvent)
	score1 := big.NewInt(0).SetBytes(ev1.Score).Uint64()
	score2 := big.NewInt(0).SetBytes(ev2.Score).Uint64()
	if score1 < score2 {
		return ev2
	}

	return ev1
}

func (p *scoreHandler) Verify(ev wire.Event) error {
	m := ev.(ScoreEvent)

	// Check first if the BidList contains valid bids
	if err := p.validateBidListSubset(m.BidListSubset); err != nil {
		return err
	}

	// Verify the proof
	seedScalar := ristretto.Scalar{}
	seedScalar.Derive(m.Seed)

	proof := zkproof.ZkProof{
		Proof:         m.Proof,
		Score:         m.Score,
		Z:             m.Z,
		BinaryBidList: m.BidListSubset,
	}

	if !proof.Verify(seedScalar) {
		return errors.New("proof verification failed")
	}

	return nil
}

func (p *scoreHandler) validateBidListSubset(bidListSubsetBytes []byte) *prerror.PrError {
	bidListSubset, err := user.ReconstructBidListSubset(bidListSubsetBytes)
	if err != nil {
		return err
	}

	p.Lock()
	defer p.Unlock()
	return p.bidList.ValidateBids(bidListSubset)
}

package generation

import (
	"bytes"
	"fmt"

	"gitlab.dusk.network/dusk-core/zkproof"

	"github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func LaunchCollectorComponent(eventBus *wire.EventBus, d, k ristretto.Scalar, bidList user.BidList) *collector {
	collector := newCollector(eventBus, d, k, bidList)
	go collector.Listen()
	return collector
}

type collector struct {
	eventBus     *wire.EventBus
	currentRound uint64
	currentStep  uint8

	*user.Keys
	d, k    ristretto.Scalar
	bidList user.BidList

	marshaller        *selection.ScoreUnMarshaller
	generator         *generator
	scoreEventChannel chan *selection.ScoreEvent
	stopChannel       chan bool

	// subscriber channels
	roundChannel   <-chan uint64
	bidListChannel <-chan user.BidList
}

func newCollector(eventBus *wire.EventBus, d, k ristretto.Scalar, bidList user.BidList) *collector {
	roundChannel := consensus.InitRoundUpdate(eventBus)
	bidListChannel := selection.InitBidListUpdate(eventBus)
	collector := &collector{
		eventBus:          eventBus,
		roundChannel:      roundChannel,
		marshaller:        &selection.ScoreUnMarshaller{},
		bidListChannel:    bidListChannel,
		bidList:           bidList,
		d:                 d,
		k:                 k,
		scoreEventChannel: make(chan *selection.ScoreEvent, 1),
		stopChannel:       make(chan bool, 1),
		generator:         newGenerator(),
	}

	go wire.NewEventSubscriber(eventBus, collector, msg.BlockGenerationTopic).Accept()
	return collector
}

func (g *collector) Listen() {
	for {
		select {
		case round := <-g.roundChannel:
			g.updateRound(round)
		case bidList := <-g.bidListChannel:
			g.bidList = bidList
		case scoreEvent := <-g.scoreEventChannel:
			// check if this score event is still relevant
			if scoreEvent.Round == g.currentRound && scoreEvent.Step == g.currentStep {
				g.sendScore(scoreEvent)
				g.currentStep++
			}
		}
	}
}

func (g *collector) generate() {
	g.stopChannel = make(chan bool, 1)
	g.generator = newGenerator()
	seed, _ := crypto.RandEntropy(33)
	go g.generator.generateProof(g.d, g.k, g.bidList, seed)
	select {
	case <-g.stopChannel:
		return
	case proof := <-g.generator.proofChannel:
		sev, err := g.generateScoreEvent(proof, seed)
		if err != nil {
			return
		}

		g.scoreEventChannel <- sev
	}
}

func (g *collector) updateRound(round uint64) {
	g.stopChannel <- true
	g.currentRound = round
	g.currentStep = 1

	go g.generate()
}

func (g *collector) generateScoreEvent(proof zkproof.ZkProof, seed []byte) (*selection.ScoreEvent, error) {
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		return nil, err
	}

	return &selection.ScoreEvent{
		Round:         g.currentRound,
		Step:          g.currentStep,
		Score:         proof.Score,
		Proof:         proof.Proof,
		Z:             proof.Z,
		BidListSubset: proof.BinaryBidList,
		Seed:          seed,
		VoteHash:      hash,
	}, nil
}

func (g *collector) sendScore(sev *selection.ScoreEvent) error {
	buffer := new(bytes.Buffer)
	topicBytes := topics.TopicToByteArray(topics.Score)
	if _, err := buffer.Write(topicBytes[:]); err != nil {
		return err
	}

	scoreBuffer := new(bytes.Buffer)
	if err := g.marshaller.Marshal(scoreBuffer, sev); err != nil {
		return err
	}

	if _, err := buffer.Write(scoreBuffer.Bytes()); err != nil {
		return err
	}

	fmt.Println("sending proof")
	// g.eventBus.Publish(string(topics.Score), scoreBuffer)
	g.eventBus.Publish(string(topics.Gossip), buffer)
	return nil
}

func (g *collector) Collect(m *bytes.Buffer) error {
	go g.generate()
	return nil
}

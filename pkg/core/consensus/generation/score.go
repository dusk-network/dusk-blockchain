package generation

import (
	"bytes"

	"github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/zkproof"
)

// LaunchScoreGenerationComponent will start the processes for score generation.
func LaunchScoreGenerationComponent(eventBus *wire.EventBus, d, k ristretto.Scalar, bidList user.BidList) *broker {
	broker := newBroker(eventBus, d, k, bidList)
	go broker.Listen()
	return broker
}

type (
	proofCollector struct {
		currentRound uint64
		currentStep  uint8

		marshaller *selection.ScoreUnMarshaller
		d, k       ristretto.Scalar
		bidList    user.BidList
		seed       []byte

		generator         generator
		scoreEventChannel chan *bytes.Buffer
		stopChannel       chan bool
	}

	broker struct {
		eventBus *wire.EventBus
		*proofCollector

		// subscriber channels
		roundChannel   <-chan uint64
		bidListChannel <-chan user.BidList
	}
)

func newProofCollector(d, k ristretto.Scalar, bidList user.BidList) *proofCollector {
	return &proofCollector{
		marshaller:        &selection.ScoreUnMarshaller{},
		d:                 d,
		k:                 k,
		bidList:           bidList,
		scoreEventChannel: make(chan *bytes.Buffer, 1),
		stopChannel:       make(chan bool, 1),
		generator:         &proofGenerator{},
	}
}

func (g *proofCollector) startGenerator() {
	g.stopChannel = make(chan bool, 1)
	proofChannel := make(chan zkproof.ZkProof, 1)
	go g.generator.generateProof(g.d, g.k, g.bidList, g.seed, proofChannel)
	g.listenGenerator(proofChannel)
}

func (g *proofCollector) listenGenerator(proofChannel chan zkproof.ZkProof) {
	select {
	case <-g.stopChannel:
		return
	case proof := <-proofChannel:
		sev, err := g.generateScoreEvent(proof)
		if err != nil {
			return
		}

		marshalledEvent := g.marshalScore(sev)
		g.scoreEventChannel <- marshalledEvent
	}
}

func (g *proofCollector) updateRound(round uint64) {
	g.stopChannel <- true
	g.currentRound = round
	g.currentStep = 1
	// TODO: make an actual seed by signing the previous block seed
	g.seed, _ = crypto.RandEntropy(33)

	go g.startGenerator()
}

func (g *proofCollector) generateScoreEvent(proof zkproof.ZkProof) (*selection.ScoreEvent, error) {
	// TODO: get an actual hash by generating a block
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
		Seed:          g.seed,
		VoteHash:      hash,
	}, nil
}

func (g *proofCollector) marshalScore(sev *selection.ScoreEvent) *bytes.Buffer {
	buffer := new(bytes.Buffer)
	if err := g.marshaller.Marshal(buffer, sev); err != nil {
		panic(err)
	}

	message, err := wire.AddTopic(buffer, topics.Score)
	if err != nil {
		panic(err)
	}

	return message
}

func newBroker(eventBus *wire.EventBus, d, k ristretto.Scalar, bidList user.BidList) *broker {
	proofCollector := newProofCollector(d, k, bidList)
	roundChannel := consensus.InitRoundUpdate(eventBus)
	bidListChannel := selection.InitBidListUpdate(eventBus)
	broker := &broker{
		eventBus:       eventBus,
		proofCollector: proofCollector,
		roundChannel:   roundChannel,
		bidListChannel: bidListChannel,
	}

	go wire.NewEventSubscriber(eventBus, broker, msg.BlockGenerationTopic).Accept()
	return broker
}

func (g *broker) Listen() {
	for {
		select {
		case round := <-g.roundChannel:
			g.updateRound(round)
		case bidList := <-g.bidListChannel:
			g.bidList = bidList
		case scoreEvent := <-g.scoreEventChannel:
			log.WithField("process", "generation").Debugln("sending proof")
			g.eventBus.Publish(string(topics.Gossip), scoreEvent)
			g.currentStep++
		}
	}
}

func (g *broker) Collect(m *bytes.Buffer) error {
	go g.startGenerator()
	return nil
}

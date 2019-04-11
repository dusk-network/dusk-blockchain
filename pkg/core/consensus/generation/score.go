package generation

import (
	"bytes"
	"sync"

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

var empty = new(struct{})

// LaunchScoreGenerationComponent will start the processes for score generation.
func LaunchScoreGenerationComponent(eventBus *wire.EventBus, d, k ristretto.Scalar) *broker {
	broker := newBroker(eventBus, d, k)
	go broker.Listen()
	return broker
}

type (
	proofCollector struct {
		state consensus.State

		marshaller *selection.ScoreUnMarshaller
		d, k       ristretto.Scalar
		bidList    user.BidList
		seed       []byte

		generator generator
		sync.RWMutex
		listener          *listener
		scoreEventChannel chan *bytes.Buffer
	}

	listener struct {
		proofChannel chan zkproof.ZkProof
		stopChannel  chan *struct{}
	}

	broker struct {
		eventBus *wire.EventBus
		*proofCollector

		// subscriber channels
		roundChannel   <-chan uint64
		bidListChannel <-chan user.BidList
	}
)

func newProofCollector(d, k ristretto.Scalar) *proofCollector {
	return &proofCollector{
		state:             consensus.NewState(),
		marshaller:        &selection.ScoreUnMarshaller{},
		d:                 d,
		k:                 k,
		scoreEventChannel: make(chan *bytes.Buffer, 1),
		RWMutex:           sync.RWMutex{},
		generator:         &proofGenerator{sync.RWMutex{}},
		listener: &listener{
			stopChannel: make(chan *struct{}, 1),
		},
	}
}

func (g *proofCollector) startGenerator() {
	g.Lock()
	g.listener.stopChannel <- empty
	g.Unlock()

	stopChannel := make(chan *struct{}, 1)
	proofChannel := make(chan zkproof.ZkProof, 1)
	go g.generator.generateProof(g.d, g.k, g.bidList, g.seed, proofChannel)
	g.listener = &listener{
		proofChannel: proofChannel,
		stopChannel:  stopChannel,
	}
	g.Lock()
	defer g.Unlock()
	g.listener.listenGenerator(g.generateScoreEvent)
}

func (l *listener) listenGenerator(generateScoreEvent func(zkproof.ZkProof)) {
	go func() {
		select {
		case <-l.stopChannel:
			return
		case proof := <-l.proofChannel:
			generateScoreEvent(proof)
		}
	}()
}

func (g *proofCollector) updateRound(round uint64) {
	g.state.Update(round)
	// TODO: make an actual seed by signing the previous block seed
	g.seed, _ = crypto.RandEntropy(33)

	g.startGenerator()
}

func (g *proofCollector) generateScoreEvent(proof zkproof.ZkProof) {
	// TODO: get an actual hash by generating a block
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		panic(err)
	}

	sev := &selection.ScoreEvent{
		Round:         g.state.Round(),
		Step:          g.state.Step(),
		Score:         proof.Score,
		Proof:         proof.Proof,
		Z:             proof.Z,
		BidListSubset: proof.BinaryBidList,
		Seed:          g.seed,
		VoteHash:      hash,
	}

	marshalledEvent := g.marshalScore(sev)
	g.scoreEventChannel <- marshalledEvent
	log.WithFields(log.Fields{
		"process":       "generation",
		"message round": sev.Round,
		"message step":  sev.Step,
	}).Debugln("created proof")
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

func newBroker(eventBus *wire.EventBus, d, k ristretto.Scalar) *broker {
	proofCollector := newProofCollector(d, k)
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
			log.WithFields(log.Fields{
				"process":         "generation",
				"collector round": g.state.Round(),
				"collector step":  g.state.Step(),
			}).Debugln("sending proof")
			g.eventBus.Publish(string(topics.Gossip), scoreEvent)
			g.state.IncrementStep()
		}
	}
}

func (g *broker) Collect(m *bytes.Buffer) error {
	g.startGenerator()
	return nil
}

package generation

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"

	"github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func LaunchGeneratorComponent(eventBus *wire.EventBus, d, k ristretto.Scalar) *generator {
	generator := newGenerator(eventBus, d, k)
	generator.Listen()
	return generator
}

type generator struct {
	eventBus     *wire.EventBus
	currentRound uint64
	currentStep  uint8

	*user.Keys
	d, k       ristretto.Scalar
	bidList    *user.BidList
	marshaller *selection.ScoreUnMarshaller

	// subscriber channels
	roundChannel   <-chan uint64
	bidListChannel <-chan *user.BidList
}

func newGenerator(eventBus *wire.EventBus, d, k ristretto.Scalar) *generator {
	roundChannel := consensus.InitRoundUpdate(eventBus)
	generator := &generator{
		eventBus:     eventBus,
		roundChannel: roundChannel,
		marshaller:   &selection.ScoreUnMarshaller{},
	}

	go wire.NewEventSubscriber(eventBus, generator, msg.BlockGenerationTopic).Accept()
	return generator
}

func (g *generator) Listen() {
	for {
		select {
		case round := <-g.roundChannel:
			g.updateRound(round)
		case bidList := <-g.bidListChannel:
			g.bidList = bidList
		}
	}
}

func (g *generator) updateRound(round uint64) {
	g.currentRound = round
	g.currentStep = 1
	sev, err := g.generateScoreEvent()
	if err != nil {
		return
	}

	if err := g.sendScore(sev); err != nil {
		return
	}

	g.currentStep++
}

func (g *generator) generateScoreEvent() (*selection.ScoreEvent, error) {
	seed, err := crypto.RandEntropy(33)
	if err != nil {
		return nil, err
	}

	proof, err := g.generateProof(seed)
	if err != nil {
		return nil, err
	}

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
		BidListSubset: proof.ProofBidList,
		Seed:          seed,
		VoteHash:      hash,
	}, nil
}

func (g *generator) sendScore(sev *selection.ScoreEvent) error {
	buffer := new(bytes.Buffer)
	topicBytes := topics.TopicToByteArray(topics.Score)
	if _, err := buffer.Write(topicBytes[:]); err != nil {
		return err
	}

	if err := g.marshaller.Marshal(buffer, sev); err != nil {
		return err
	}

	g.eventBus.Publish(string(topics.Gossip), buffer)
	return nil
}

func (g *generator) Collect(m *bytes.Buffer) error {
	sev, err := g.generateScoreEvent()
	if err != nil {
		return err
	}

	if err := g.sendScore(sev); err != nil {
		return err
	}

	g.currentStep++
	return nil
}

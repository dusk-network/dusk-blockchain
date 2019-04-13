package generation

import (
	"bytes"

	"github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/zkproof"
)

// LaunchScoreGenerationComponent will start the processes for score generation.
func LaunchScoreGenerationComponent(eventBus *wire.EventBus, d, k ristretto.Scalar) *broker {
	broker := newBroker(eventBus, d, k)
	go broker.Listen()
	return broker
}

type broker struct {
	proofGenerator *proofGenerator
	forwarder      *forwarder
	seeder         *seeder

	// subscriber channels
	roundChannel   <-chan uint64
	bidListChannel <-chan user.BidList
}

func newBroker(eventBroker wire.EventBroker, d, k ristretto.Scalar) *broker {
	proofGenerator := newProofGenerator(d, k)
	roundChannel := consensus.InitRoundUpdate(eventBroker)
	bidListChannel := consensus.InitBidListUpdate(eventBroker)
	broker := &broker{
		proofGenerator: proofGenerator,
		roundChannel:   roundChannel,
		bidListChannel: bidListChannel,
		forwarder:      newForwarder(eventBroker),
		seeder:         &seeder{},
	}

	go wire.NewTopicListener(eventBroker, broker, msg.BlockRegenerationTopic).Accept()
	return broker
}

func (b *broker) Listen() {
	for {
		select {
		case round := <-b.roundChannel:
			seed := b.seeder.GenerateSeed(round)
			proof := b.proofGenerator.generateProof(seed)
			b.Forward(proof, seed)
		case bidList := <-b.bidListChannel:
			b.proofGenerator.updateBidList(bidList)
		}
	}
}

func (b *broker) Collect(m *bytes.Buffer) error {
	seed := b.seeder.LatestSeed()
	proof := b.proofGenerator.generateProof(seed)
	b.Forward(proof, seed)
	return nil
}

func (b *broker) Forward(proof zkproof.ZkProof, seed []byte) {
	b.seeder.RLock()
	if b.seeder.isFresh(seed) {
		round := b.seeder.round
		b.seeder.RUnlock()
		b.forwarder.forwardScoreEvent(proof, round, seed)
		return
	}

	b.seeder.RUnlock()
}

package generation

import (
	"github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
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
	proofGenerator generator
	forwarder      *forwarder
	seeder         *seeder

	// subscriber channels
	roundChan        <-chan uint64
	bidListChan      <-chan user.BidList
	regenerationChan <-chan consensus.AsyncState
}

func newBroker(eventBroker wire.EventBroker, d, k ristretto.Scalar) *broker {
	proofGenerator := newProofGenerator(d, k)
	roundChan := consensus.InitRoundUpdate(eventBroker)
	bidListChan := consensus.InitBidListUpdate(eventBroker)
	regenerationChan := consensus.InitBlockRegenerationCollector(eventBroker)
	return &broker{
		proofGenerator:   proofGenerator,
		roundChan:        roundChan,
		bidListChan:      bidListChan,
		regenerationChan: regenerationChan,
		forwarder:        newForwarder(eventBroker),
		seeder:           &seeder{},
	}
}

func (b *broker) Listen() {
	for {
		select {
		case round := <-b.roundChan:
			seed := b.seeder.GenerateSeed(round)
			proof := b.proofGenerator.generateProof(seed)
			b.Forward(proof, seed)
		case bidList := <-b.bidListChan:
			b.proofGenerator.updateBidList(bidList)
		case state := <-b.regenerationChan:
			if state.Round == b.seeder.Round() {
				seed := b.seeder.LatestSeed()
				proof := b.proofGenerator.generateProof(seed)
				b.Forward(proof, seed)
			}
		}
	}
}

func (b *broker) Forward(proof zkproof.ZkProof, seed []byte) {
	if b.seeder.isFresh(seed) {
		b.forwarder.forwardScoreEvent(proof, b.seeder.Round(), seed)
	}
}

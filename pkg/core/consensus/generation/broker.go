package generation

import (
	"github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/zkproof"
)

// LaunchScoreGenerationComponent will start the processes for score generation.
func LaunchScoreGenerationComponent(eventBus *wire.EventBus, rpcBus *wire.RPCBus,
	d, k ristretto.Scalar, gen Generator, blockGen BlockGenerator) *broker {
	broker := newBroker(eventBus, rpcBus, d, k, gen, blockGen)
	go broker.Listen()
	return broker
}

type broker struct {
	proofGenerator Generator
	forwarder      *forwarder
	seeder         *seeder

	// subscriber channels
	roundChan        <-chan uint64
	bidListChan      <-chan user.BidList
	regenerationChan <-chan consensus.AsyncState
}

func newBroker(eventBroker wire.EventBroker, rpcBus *wire.RPCBus, d, k ristretto.Scalar,
	gen Generator, blockGen BlockGenerator) *broker {
	if gen == nil {
		gen = newProofGenerator(d, k)
	}

	if blockGen == nil {
		blockGen = newBlockGenerator(rpcBus)
	}

	roundChan := consensus.InitRoundUpdate(eventBroker)
	bidListChan := consensus.InitBidListUpdate(eventBroker)
	regenerationChan := consensus.InitBlockRegenerationCollector(eventBroker)
	return &broker{
		proofGenerator:   gen,
		roundChan:        roundChan,
		bidListChan:      bidListChan,
		regenerationChan: regenerationChan,
		forwarder:        newForwarder(eventBroker, blockGen),
		seeder:           &seeder{},
	}
}

func (b *broker) Listen() {
	for {
		select {
		case round := <-b.roundChan:
			seed := b.seeder.GenerateSeed(round)
			proof := b.proofGenerator.GenerateProof(seed)
			b.Forward(proof, seed)
		case bidList := <-b.bidListChan:
			b.proofGenerator.UpdateBidList(bidList)
		case state := <-b.regenerationChan:
			if state.Round == b.seeder.Round() {
				seed := b.seeder.LatestSeed()
				proof := b.proofGenerator.GenerateProof(seed)
				b.Forward(proof, seed)
			}
		}
	}
}

func (b *broker) Forward(proof zkproof.ZkProof, seed []byte) {
	if b.seeder.isFresh(seed) {
		if err := b.forwarder.forwardScoreEvent(proof, b.seeder.Round(), seed); err != nil {
			log.WithFields(log.Fields{
				"process": "generation",
			}).WithError(err).Errorln("error forwarding score event")
		}
	}
}

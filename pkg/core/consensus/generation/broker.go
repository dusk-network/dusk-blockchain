package generation

import (
	"crypto/rand"

	"github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/zkproof"
)

// Launch will start the processes for score/block generation.
func Launch(eventBus *wire.EventBus, rpcBus *wire.RPCBus,
	d, k ristretto.Scalar, gen Generator, blockGen BlockGenerator) {
	broker := newBroker(eventBus, rpcBus, d, k, gen, blockGen)
	go broker.Listen()
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

	seed := make([]byte, 64)
	_, _ = rand.Read(seed)

	// TODO: Read Block Generator's PublicKey from config or dusk-wallet API
	publicKey := key.NewKeyPair(seed).PublicKey()

	if blockGen == nil {
		blockGen = newBlockGenerator(publicKey, rpcBus)
	}

	roundChan := consensus.InitRoundUpdate(eventBroker)
	bidListChan := consensus.InitBidListUpdate(eventBroker)
	regenerationChan := consensus.InitBlockRegenerationCollector(eventBroker)

	return &broker{
		proofGenerator:   gen,
		roundChan:        roundChan,
		bidListChan:      bidListChan,
		regenerationChan: regenerationChan,
		forwarder:        newForwarder(eventBroker, blockGen, rpcBus),
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
			b.forwarder.threshold.Lower()
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

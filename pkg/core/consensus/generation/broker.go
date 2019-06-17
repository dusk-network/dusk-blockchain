package generation

import (
	"crypto/rand"

	"github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
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
	bidListChan       <-chan user.BidList
	regenerationChan  <-chan consensus.AsyncState
	acceptedBlockChan <-chan block.Block
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

	bidListChan := consensus.InitBidListUpdate(eventBroker)
	regenerationChan := consensus.InitBlockRegenerationCollector(eventBroker)
	acceptedBlockChan := consensus.InitAcceptedBlockUpdate(eventBroker)

	return &broker{
		proofGenerator:    gen,
		bidListChan:       bidListChan,
		regenerationChan:  regenerationChan,
		acceptedBlockChan: acceptedBlockChan,
		forwarder:         newForwarder(eventBroker, blockGen, rpcBus),
		seeder:            &seeder{},
	}
}

func (b *broker) Listen() {
	for {
		select {
		case bidList := <-b.bidListChan:
			b.proofGenerator.UpdateBidList(bidList)
		case state := <-b.regenerationChan:
			b.forwarder.threshold.Lower()
			if state.Round == b.seeder.Round() {
				seed := b.seeder.LatestSeed()
				proof := b.proofGenerator.GenerateProof(seed)
				b.Forward(proof, seed)
			}
		case acceptedBlock := <-b.acceptedBlockChan:
			b.forwarder.blockGenerator.UpdatePrevBlock(acceptedBlock)
			seed := b.seeder.GenerateSeed(acceptedBlock.Header.Height + 1)
			proof := b.proofGenerator.GenerateProof(seed)
			b.Forward(proof, seed)
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

package generation

import (
	"bytes"
	"crypto/rand"

	"github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/zkproof"
)

// Launch will start the processes for score/block generation.
func Launch(eventBus *wire.EventBus, rpcBus *wire.RPCBus, d, k ristretto.Scalar, gen Generator, blockGen BlockGenerator, keys user.Keys) {
	broker := newBroker(eventBus, rpcBus, d, k, gen, blockGen, keys)
	go broker.Listen()
}

type broker struct {
	publisher            wire.EventPublisher
	proofGenerator       Generator
	forwarder            *forwarder
	seeder               *seeder
	certificateGenerator *certificateGenerator

	// subscriber channels
	bidListChan          <-chan user.BidList
	regenerationChan     <-chan consensus.AsyncState
	winningBlockHashChan <-chan []byte
	acceptedBlockChan    <-chan block.Block
}

func newBroker(eventBroker wire.EventBroker, rpcBus *wire.RPCBus, d, k ristretto.Scalar,
	gen Generator, blockGen BlockGenerator, keys user.Keys) *broker {
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

	certGenerator := &certificateGenerator{}
	eventBroker.SubscribeCallback(msg.AgreementEventTopic, certGenerator.setAgreementEvent)

	return &broker{
		publisher:            eventBroker,
		proofGenerator:       gen,
		certificateGenerator: certGenerator,
		bidListChan:          consensus.InitBidListUpdate(eventBroker),
		regenerationChan:     consensus.InitBlockRegenerationCollector(eventBroker),
		winningBlockHashChan: initWinningHashCollector(eventBroker),
		acceptedBlockChan:    consensus.InitAcceptedBlockUpdate(eventBroker),
		forwarder:            newForwarder(eventBroker, blockGen, rpcBus),
		seeder:               &seeder{keys: keys},
	}
}

func (b *broker) Listen() {
	for {
		select {
		case bidList := <-b.bidListChan:
			b.proofGenerator.UpdateBidList(bidList)
		case state := <-b.regenerationChan:
			if state.Round == b.seeder.Round() {
				b.forwarder.threshold.Lower()
				b.generateProofAndBlock()
			}
		case winningBlockHash := <-b.winningBlockHashChan:
			cert := b.certificateGenerator.generateCertificate()
			b.sendCertificateMsg(cert, winningBlockHash)
		case acceptedBlock := <-b.acceptedBlockChan:
			b.onBlock(acceptedBlock)
		}
	}
}

func (b *broker) onBlock(blk block.Block) {
	// Remove old bids before generating a new score
	b.proofGenerator.RemoveExpiredBids(blk.Header.Height + 1)

	if err := b.seeder.GenerateSeed(blk.Header.Height+1, blk.Header.Seed); err != nil {
		log.WithField("process", "generation").WithError(err).Errorln("problem generating seed")
	}

	b.forwarder.setPrevBlock(blk)
	b.generateProofAndBlock()
}

func (b *broker) generateProofAndBlock() {
	proof := b.proofGenerator.GenerateProof(b.seeder.LatestSeed())
	b.Forward(proof, b.seeder.LatestSeed())
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

func (b *broker) sendCertificateMsg(cert *block.Certificate, blockHash []byte) error {
	buf := new(bytes.Buffer)
	if err := encoding.Write256(buf, blockHash); err != nil {
		return err
	}

	if err := cert.Encode(buf); err != nil {
		return err
	}

	b.publisher.Publish(string(topics.Certificate), buf)
	return nil
}

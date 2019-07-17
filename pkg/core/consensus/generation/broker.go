package generation

import (
	"bytes"
	"crypto/rand"

	"github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/zkproof"
)

// Launch will start the processes for score/block generation.
func Launch(eventBus wire.EventBroker, rpcBus *wire.RPCBus, d, k ristretto.Scalar, gen Generator, blockGen BlockGenerator, keys user.Keys, publicKey *key.PublicKey) {
	broker := newBroker(eventBus, rpcBus, d, k, gen, blockGen, keys, publicKey)
	go broker.Listen()
}

type broker struct {
	publisher            wire.EventPublisher
	proofGenerator       Generator
	forwarder            *forwarder
	seeder               *seeder
	certificateGenerator *certificateGenerator

	// subscriber channels
	bidChan              <-chan user.Bid
	regenerationChan     <-chan consensus.AsyncState
	winningBlockHashChan <-chan []byte
}

func newBroker(eventBroker wire.EventBroker, rpcBus *wire.RPCBus, d, k ristretto.Scalar,
	gen Generator, blockGen BlockGenerator, keys user.Keys, publicKey *key.PublicKey) *broker {
	if gen == nil {
		gen = newProofGenerator(d, k)
	}

	seed := make([]byte, 64)
	_, _ = rand.Read(seed)

	if blockGen == nil {
		blockGen = newBlockGenerator(publicKey, rpcBus)
	}

	blk := getLatestBlock()
	certGenerator := &certificateGenerator{}
	eventBroker.SubscribeCallback(msg.AgreementEventTopic, certGenerator.setAgreementEvent)

	b := &broker{
		publisher:            eventBroker,
		proofGenerator:       gen,
		certificateGenerator: certGenerator,
		bidChan:              consensus.InitBidListUpdate(eventBroker),
		regenerationChan:     consensus.InitBlockRegenerationCollector(eventBroker),
		winningBlockHashChan: initWinningHashCollector(eventBroker),
		forwarder:            newForwarder(eventBroker, blockGen),
		seeder:               &seeder{keys: keys},
	}
	eventBroker.SubscribeCallback(string(topics.AcceptedBlock), b.onBlock)
	b.handleBlock(blk)
	return b
}

func getLatestBlock() *block.Block {
	_, db := heavy.CreateDBConnection()
	var blk *block.Block
	err := db.View(func(t database.Transaction) error {
		currentHeight, err := t.FetchCurrentHeight()
		if err != nil {
			return err
		}

		hash, err := t.FetchBlockHashByHeight(currentHeight)
		if err != nil {
			return err
		}

		blk, err = t.FetchBlock(hash)
		return err
	})

	if err != nil {
		return config.DecodeGenesis()
	}

	return blk
}

func (b *broker) Listen() {
	for {
		select {
		case bid := <-b.bidChan:
			b.proofGenerator.UpdateBidList(bid)
		case state := <-b.regenerationChan:
			if state.Round == b.seeder.Round() {
				b.forwarder.threshold.Lower()
				b.generateProofAndBlock()
			}
		case winningBlockHash := <-b.winningBlockHashChan:
			cert := b.certificateGenerator.generateCertificate()
			b.sendCertificateMsg(cert, winningBlockHash)
		}
	}
}

func (b *broker) onBlock(m *bytes.Buffer) error {
	b.forwarder.threshold.Reset()

	blk := block.NewBlock()
	if err := blk.Decode(m); err != nil {
		return err
	}

	// Remove old bids before generating a new score
	b.proofGenerator.RemoveExpiredBids(blk.Header.Height + 1)

	return b.handleBlock(blk)
}

func (b *broker) handleBlock(blk *block.Block) error {
	if err := b.seeder.GenerateSeed(blk.Header.Height+1, blk.Header.Seed); err != nil {
		return err
	}

	b.forwarder.setPrevBlock(*blk)
	b.generateProofAndBlock()
	return nil
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
			}).WithError(err).Warnln("error forwarding score event")
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

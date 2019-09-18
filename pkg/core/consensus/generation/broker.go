package generation

import (
	"bytes"
	"crypto/rand"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-wallet/key"
	zkproof "github.com/dusk-network/dusk-zkproof"
	log "github.com/sirupsen/logrus"
)

// Launch will start the processes for score/block generation.
func Launch(eventBus wire.EventBroker, rpcBus *wire.RPCBus, k ristretto.Scalar, keys user.Keys, publicKey *key.PublicKey, gen Generator, blockGen BlockGenerator, db database.DB) error {
	m := zkproof.CalculateM(k)
	d := getD(m, eventBus, db)
	broker, err := newBroker(eventBus, rpcBus, d, k, m, gen, blockGen, keys, publicKey)
	if err != nil {
		return err
	}

	go broker.Listen()
	return nil
}

type broker struct {
	k              ristretto.Scalar
	m              ristretto.Scalar
	eventBroker    wire.EventBroker
	proofGenerator Generator

	forwarder            *forwarder
	seeder               *seeder
	certificateGenerator *certificateGenerator

	// subscriber channels
	regenerationChan     <-chan consensus.AsyncState
	winningBlockHashChan <-chan []byte
	acceptedBlockChan    <-chan block.Block
}

func newBroker(eventBroker wire.EventBroker, rpcBus *wire.RPCBus, d, k, m ristretto.Scalar,
	gen Generator, blockGen BlockGenerator, keys user.Keys, publicKey *key.PublicKey) (*broker, error) {
	if gen == nil {
		var err error
		gen, err = newProofGenerator(d, k, m)
		if err != nil {
			return nil, err
		}
	}

	seed := make([]byte, 64)
	_, _ = rand.Read(seed)

	if blockGen == nil {
		blockGen = newBlockGenerator(publicKey, rpcBus)
	}

	blk := getLatestBlock()
	certGenerator := &certificateGenerator{}
	eventBroker.SubscribeCallback(msg.AgreementEventTopic, certGenerator.setAgreementEvent)
	acceptedBlockChan, _ := consensus.InitAcceptedBlockUpdate(eventBroker)

	b := &broker{
		k:                    k,
		m:                    m,
		eventBroker:          eventBroker,
		proofGenerator:       gen,
		certificateGenerator: certGenerator,
		regenerationChan:     consensus.InitBlockRegenerationCollector(eventBroker),
		winningBlockHashChan: initWinningHashCollector(eventBroker),
		acceptedBlockChan:    acceptedBlockChan,
		forwarder:            newForwarder(eventBroker, blockGen),
		seeder:               &seeder{keys: keys},
	}
	b.handleBlock(*blk)
	return b, nil
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
		case state := <-b.regenerationChan:
			if state.Round == b.seeder.Round() {
				b.forwarder.threshold.Lower()
				b.generateProofAndBlock()
			}
		case winningBlockHash := <-b.winningBlockHashChan:
			cert := b.certificateGenerator.generateCertificate()
			b.sendCertificateMsg(cert, winningBlockHash)
		case blk := <-b.acceptedBlockChan:
			b.onBlock(blk)
		}
	}
}

func (b *broker) onBlock(blk block.Block) error {
	b.forwarder.threshold.Reset()

	return b.handleBlock(blk)
}

func (b *broker) handleBlock(blk block.Block) error {
	if err := b.seeder.GenerateSeed(blk.Header.Height+1, blk.Header.Seed); err != nil {
		return err
	}

	b.forwarder.setPrevBlock(blk)

	// Only generate if we are allowed to
	if b.proofGenerator.InBidList() {
		b.generateProofAndBlock()
	} else {
		// Then, we will wait till a new bid belonging to us is found
		b.updateProofValues()

		// Drain the regeneration, winning block hash and accepted block channels
		b.drainChannels()
	}

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

	if err := block.MarshalCertificate(buf, cert); err != nil {
		return err
	}

	b.eventBroker.Publish(string(topics.Certificate), buf)
	return nil
}

func (b *broker) updateProofValues() {
	d := getD(b.m, b.eventBroker, nil)
	log.WithFields(log.Fields{
		"process": "generation",
	}).Debugln("changing d value")

	b.proofGenerator.UpdateProofValues(d, b.m)
}

func (b *broker) drainChannels() {
	for len(b.regenerationChan) > 0 {
		<-b.regenerationChan
	}

	for len(b.winningBlockHashChan) > 0 {
		<-b.winningBlockHashChan
	}

	for len(b.acceptedBlockChan) > 0 {
		<-b.acceptedBlockChan
	}
}

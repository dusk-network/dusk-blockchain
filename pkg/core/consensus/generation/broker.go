package generation

import (
	"bytes"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/key"
	zkproof "github.com/dusk-network/dusk-zkproof"
	log "github.com/sirupsen/logrus"
)

// Launch will start the processes for score/block generation.
func Launch(eventBus eventbus.Broker, rpcBus *rpcbus.RPCBus, k ristretto.Scalar, keys user.Keys, publicKey *key.PublicKey, gen Generator, blockGen BlockGenerator, db database.DB) error {
	broker, err := newBroker(eventBus, rpcBus, k, gen, blockGen, keys, publicKey)
	if err != nil {
		return err
	}

	state := NewState(eventBus)
	state.Wire(broker)
	return nil
}

type broker struct {
	k           ristretto.Scalar
	m           ristretto.Scalar
	eventBroker eventbus.Broker
	blockGen    BlockGenerator

	certificateGenerator *certificateGenerator

	// subscriber channels
	winningBlockHashChan <-chan []byte
}

func newBroker(eventBroker eventbus.Broker, rpcBus *rpcbus.RPCBus, k ristretto.Scalar,
	gen Generator, blockGen BlockGenerator, keys user.Keys, publicKey *key.PublicKey) (*broker, error) {
	if gen == nil {
		var err error
		gen, err = newProofGenerator(k)
		if err != nil {
			return nil, err
		}
	}

	if blockGen == nil {
		blockGen = newBlockGenerator(publicKey, rpcBus, gen, keys)
	}

	certGenerator := &certificateGenerator{}
	cbListener := eventbus.NewCallbackListener(certGenerator.setAgreementEvent)
	eventBroker.Subscribe(topics.AgreementEvent, cbListener)

	m := zkproof.CalculateM(k)
	b := &broker{
		k:                    k,
		m:                    m,
		eventBroker:          eventBroker,
		blockGen:             blockGen,
		certificateGenerator: certGenerator,
		winningBlockHashChan: initWinningHashCollector(eventBroker),
	}
	return b, nil
}

func (b *broker) Listen() {
	for {
		winningBlockHash := <-b.winningBlockHashChan
		cert := b.certificateGenerator.generateCertificate()
		b.sendCertificateMsg(cert, winningBlockHash)
	}
}

func (b *broker) Generate(roundUpdate consensus.RoundUpdate) {
	blk, sev, err := b.blockGen.Generate(roundUpdate)
	if err == bidNotFound {
		d := getD(b.m, b.eventBroker, nil)
		log.WithFields(log.Fields{
			"process": "generation",
		}).Debugln("changing d value")
		b.blockGen.UpdateProofValues(d, b.m)
		b.drainChannels()
		return
	}
	if err != nil {
		log.WithFields(log.Fields{
			"process": "generation",
		}).WithError(err).Warnln("error generating proof and block")
		return
	}

	marshalledEvent := b.marshalScore(sev)
	marshalledBlock := b.marshalBlock(blk)
	b.eventBroker.Publish(topics.Gossip, marshalledEvent)
	b.eventBroker.Publish(topics.Gossip, marshalledBlock)
}

func (b *broker) sendCertificateMsg(cert *block.Certificate, blockHash []byte) error {
	buf := new(bytes.Buffer)
	if err := encoding.Write256(buf, blockHash); err != nil {
		return err
	}

	if err := block.MarshalCertificate(buf, cert); err != nil {
		return err
	}

	b.eventBroker.Publish(topics.Certificate, buf)
	return nil
}

func (b *broker) drainChannels() {
	for len(b.winningBlockHashChan) > 0 {
		<-b.winningBlockHashChan
	}
}

func (b *broker) marshalScore(sev selection.ScoreEvent) *bytes.Buffer {
	buffer := new(bytes.Buffer)
	if err := selection.MarshalScoreEvent(buffer, &sev); err != nil {
		panic(err)
	}

	// XXX: uh? the buffer is locally defined. Why do we propagate a copy of it?
	copy := *buffer
	b.eventBroker.Publish(topics.Score, &copy)

	if err := topics.Prepend(buffer, topics.Score); err != nil {
		panic(err)
	}

	return buffer
}

func (b *broker) marshalBlock(blk block.Block) *bytes.Buffer {
	buffer := new(bytes.Buffer)
	if err := block.Marshal(buffer, &blk); err != nil {
		panic(err)
	}

	copy := *buffer
	b.eventBroker.Publish(topics.Candidate, &copy)
	if err := topics.Prepend(buffer, topics.Candidate); err != nil {
		panic(err)
	}

	return buffer
}

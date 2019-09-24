package generation

import (
	"bytes"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
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
	m := zkproof.CalculateM(k)
	d := getD(m, eventBus, db)
	broker, err := newBroker(eventBus, rpcBus, d, k, m, gen, blockGen, keys, publicKey)
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
	regenerationChan     <-chan consensus.AsyncState
	winningBlockHashChan <-chan []byte
	roundChan            <-chan consensus.RoundUpdate
}

func newBroker(eventBroker eventbus.Broker, rpcBus *rpcbus.RPCBus, d, k, m ristretto.Scalar,
	gen Generator, blockGen BlockGenerator, keys user.Keys, publicKey *key.PublicKey) (*broker, error) {
	if gen == nil {
		var err error
		gen, err = newProofGenerator(d, k, m)
		if err != nil {
			return nil, err
		}
	}

	if blockGen == nil {
		blockGen = newBlockGenerator(publicKey, rpcBus, gen, keys)
	}

	certGenerator := &certificateGenerator{}
	eventBroker.SubscribeCallback(msg.AgreementEventTopic, certGenerator.setAgreementEvent)

	b := &broker{
		k:                    k,
		m:                    m,
		eventBroker:          eventBroker,
		blockGen:             blockGen,
		certificateGenerator: certGenerator,
		regenerationChan:     consensus.InitBlockRegenerationCollector(eventBroker),
		winningBlockHashChan: initWinningHashCollector(eventBroker),
		roundChan:            consensus.InitRoundUpdate(eventBroker),
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
	b.eventBroker.Stream(string(topics.Gossip), *marshalledEvent)
	b.eventBroker.Stream(string(topics.Gossip), *marshalledBlock)
}

func (b *broker) sendCertificateMsg(cert *block.Certificate, blockHash []byte) error {
	buf := new(bytes.Buffer)
	if err := encoding.Write256(buf, blockHash); err != nil {
		return err
	}

	if err := block.MarshalCertificate(buf, cert); err != nil {
		return err
	}

	b.eventBroker.Publish(string(topics.Certificate), *buf)
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

	b.eventBroker.Publish(string(topics.Score), *buffer)
	message, err := wire.AddTopic(buffer, topics.Score)
	if err != nil {
		panic(err)
	}

	return message
}

func (b *broker) marshalBlock(blk block.Block) *bytes.Buffer {
	buffer := new(bytes.Buffer)
	if err := block.Marshal(buffer, &blk); err != nil {
		panic(err)
	}

	b.eventBroker.Publish(string(topics.Candidate), *buffer)
	message, err := wire.AddTopic(buffer, topics.Candidate)
	if err != nil {
		panic(err)
	}

	return message
}

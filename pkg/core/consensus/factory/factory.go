package factory

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/candidate"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/generation"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reputation"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type initCollector struct {
	initChannel chan uint64
}

func (i *initCollector) Collect(roundBuffer *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	i.initChannel <- round
	return nil
}

// ConsensusFactory is responsible for initializing the consensus processes
// with the proper parameters. It subscribes to the initialization topic and,
// upon reception of a message, will start all of the components related to
// consensus. It should also contain all the relevant information for the
// processes it intends to start up.
type ConsensusFactory struct {
	eventBus    *wire.EventBus
	rpcBus      *wire.RPCBus
	initChannel chan uint64

	user.Keys
	timerLength time.Duration
	d, k        ristretto.Scalar
}

// New returns an initialized ConsensusFactory.
func New(eventBus *wire.EventBus, rpcBus *wire.RPCBus, timerLength time.Duration,
	keys user.Keys, d, k ristretto.Scalar) *ConsensusFactory {
	initChannel := make(chan uint64, 1)

	initCollector := &initCollector{initChannel}
	go wire.NewTopicListener(eventBus, initCollector, msg.InitializationTopic).Accept()

	return &ConsensusFactory{
		eventBus:    eventBus,
		rpcBus:      rpcBus,
		initChannel: initChannel,
		Keys:        keys,
		timerLength: timerLength,
		d:           d,
		k:           k,
	}
}

// StartConsensus will wait for a message to come in, and then proceed to
// start the consensus components.
func (c *ConsensusFactory) StartConsensus() {
	log.WithField("process", "factory").Info("Starting consensus")
	reputation.Launch(c.eventBus)
	generation.Launch(c.eventBus, c.rpcBus, c.d, c.k, nil, nil)
	candidate.Launch(c.eventBus)
	selection.Launch(c.eventBus, c.timerLength)
	reduction.Launch(c.eventBus, nil, c.Keys, c.timerLength)

	// Wait for the initial round to be published
	round := <-c.initChannel
	log.WithFields(log.Fields{
		"process": "factory",
		"round":   round,
	}).Debug("Received initial round")

	agreement.Launch(c.eventBus, nil, c.Keys, round)
	log.WithField("process", "factory").Info("Consensus Started")
}

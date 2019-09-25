package factory

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/key"
	log "github.com/sirupsen/logrus"
)

// ConsensusFactory is responsible for initializing the consensus processes
// with the proper parameters. It subscribes to the initialization topic and,
// upon reception of a message, will start all of the components related to
// consensus. It should also contain all the relevant information for the
// processes it intends to start up.
type ConsensusFactory struct {
	eventBus eventbus.Broker
	rpcBus   *rpcbus.RPCBus

	key.ConsensusKeys
	timerLength time.Duration
}

// New returns an initialized ConsensusFactory.
func New(eventBus eventbus.Broker, rpcBus *rpcbus.RPCBus, timerLength time.Duration,
	keys key.ConsensusKeys) *ConsensusFactory {

	return &ConsensusFactory{
		eventBus:      eventBus,
		rpcBus:        rpcBus,
		ConsensusKeys: keys,
		timerLength:   timerLength,
	}
}

// StartConsensus will wait for a message to come in, and then proceed to
// start the consensus components.
func (c *ConsensusFactory) StartConsensus() {
	log.WithField("process", "factory").Info("Starting consensus")
	selection.Launch(c.eventBus, nil, c.timerLength)
	reduction.Launch(c.eventBus, c.ConsensusKeys, c.timerLength, c.rpcBus)
	go agreement.Launch(c.eventBus, c.ConsensusKeys)
	log.WithField("process", "factory").Info("Consensus Started")
}

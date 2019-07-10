package factory

import (
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reputation"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// ConsensusFactory is responsible for initializing the consensus processes
// with the proper parameters. It subscribes to the initialization topic and,
// upon reception of a message, will start all of the components related to
// consensus. It should also contain all the relevant information for the
// processes it intends to start up.
type ConsensusFactory struct {
	eventBus wire.EventBroker
	rpcBus   *wire.RPCBus

	user.Keys
	timerLength time.Duration
}

// New returns an initialized ConsensusFactory.
func New(eventBus wire.EventBroker, rpcBus *wire.RPCBus, timerLength time.Duration,
	keys user.Keys) *ConsensusFactory {

	return &ConsensusFactory{
		eventBus:    eventBus,
		rpcBus:      rpcBus,
		Keys:        keys,
		timerLength: timerLength,
	}
}

// StartConsensus will wait for a message to come in, and then proceed to
// start the consensus components.
func (c *ConsensusFactory) StartConsensus() {
	log.WithField("process", "factory").Info("Starting consensus")
	reputation.Launch(c.eventBus)
	selection.Launch(c.eventBus, nil, c.timerLength)
	reduction.Launch(c.eventBus, nil, c.Keys, c.timerLength, c.rpcBus)
	go agreement.Launch(c.eventBus, nil, c.Keys)
	log.WithField("process", "factory").Info("Consensus Started")
}

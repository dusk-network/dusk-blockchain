package factory

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/chain/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation/score"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction/firststep"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction/secondstep"
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
	eventBus *eventbus.EventBus
	rpcBus   *rpcbus.RPCBus

	walletPubKey *key.PublicKey
	key.ConsensusKeys
	timerLength time.Duration
}

// New returns an initialized ConsensusFactory.
func New(eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, timerLength time.Duration, walletPubKey *key.PublicKey, keys key.ConsensusKeys) *ConsensusFactory {
	return &ConsensusFactory{
		eventBus:      eventBus,
		rpcBus:        rpcBus,
		walletPubKey:  walletPubKey,
		ConsensusKeys: keys,
		timerLength:   timerLength,
	}
}

// StartConsensus will wait for a message to come in, and then proceed to
// start the consensus components.
func (c *ConsensusFactory) StartConsensus() {
	log.WithField("process", "factory").Info("Starting consensus")
	gen := generation.NewFactory()
	cgen := candidate.NewFactory(c.eventBus, c.rpcBus, c.walletPubKey)
	sgen := score.NewFactory(c.eventBus, c.ConsensusKeys, nil)
	sel := selection.NewFactory(c.eventBus, c.timerLength)
	redFirstStep := firststep.NewFactory(c.eventBus, c.rpcBus, c.ConsensusKeys, c.timerLength)
	redSecondStep := secondstep.NewFactory(c.eventBus, c.rpcBus, c.ConsensusKeys, c.timerLength)
	agr := agreement.NewFactory(c.eventBus, c.ConsensusKeys)

	consensus.Start(c.eventBus, c.ConsensusKeys, cgen, sgen, sel, redFirstStep, redSecondStep, agr, gen)
	log.WithField("process", "factory").Info("Consensus Started")
}

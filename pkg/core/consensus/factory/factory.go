package factory

import (
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation/score"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction/firststep"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction/secondstep"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
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

	pubKey *transactions.PublicKey
	key.Keys
	timerLength time.Duration

	proxy transactions.Proxy
	ctx   context.Context
}

// New returns an initialized ConsensusFactory.
func New(ctx context.Context, eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, timerLength time.Duration, pubKey *transactions.PublicKey, keys key.Keys, proxy transactions.Proxy) *ConsensusFactory {
	return &ConsensusFactory{
		eventBus:    eventBus,
		rpcBus:      rpcBus,
		pubKey:      pubKey,
		Keys:        keys,
		timerLength: timerLength,
		proxy:       proxy,
		ctx:         ctx,
	}
}

// StartConsensus will wait for a message to come in, and then proceed to
// start the consensus components.
func (c *ConsensusFactory) StartConsensus() {
	log.WithField("process", "factory").Info("Starting consensus")
	gen := generation.NewFactory()
	cgen := candidate.NewFactory(c.ctx, c.eventBus, c.rpcBus, c.pubKey)
	sgen := score.NewFactory(c.ctx, c.eventBus, c.Keys, nil, c.proxy)
	sel := selection.NewFactory(c.ctx, c.eventBus, c.timerLength, c.proxy)
	redFirstStep := firststep.NewFactory(c.eventBus, c.rpcBus, c.Keys, c.timerLength)
	redSecondStep := secondstep.NewFactory(c.eventBus, c.rpcBus, c.Keys, c.timerLength)
	agr := agreement.NewFactory(c.eventBus, c.Keys)

	consensus.Start(c.eventBus, c.Keys, cgen, sgen, sel, redFirstStep, redSecondStep, agr, gen)
	log.WithField("process", "factory").Info("Consensus Started")
}

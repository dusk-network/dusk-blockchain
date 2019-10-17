package generation

import (
	"bytes"
	"time"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

func newComponent(publisher eventbus.Publisher, rpcBus *rpcbus.RPCBus, keys user.Keys, timeOut time.Duration) *generator {
	return &generator{
		publisher: publisher,
		rpcBus:    rpcBus,
		keys:      keys,
	}
}

type generator struct {
	store     consensus.Store
	k         ristretto.Scalar
	publisher eventbus.Publisher
	rpcBus    *rpcbus.RPCBus
	bidList   user.BidList

	blockGen BlockGenerator
}

// Initialize the generator, by creating listeners for the desired topics.
// Implements consensus.Component.
func (g *generator) Initialize(store consensus.Store, ru consensus.RoundUpdate) []consensus.Subscriber {
	g.store = store
	g.bidList = ru.BidList

	// If we are not in this round's bid list, we can skip initialization, as there
	// would be no need to listen for these events if we are not qualified to generate
	// scores and blocks.
	if !b.blockGen.proofGenerator.InBidList(g.bidList) {
		return nil
	}

	regenSubscriber := consensus.Subscriber{
		Topic:    topics.Regeneration,
		Listener: consensus.NewSimpleListener(s.CollectRegeneration),
	}

	agreementEventSubscriber := consensus.Subscriber{
		Topic:    topics.AgreementEvent,
		Listener: consensus.NewSimpleListener(g.CollectAgreementEvent),
	}

	// We always generate a block and score immediately on round update.
	g.generateBlock()
	return []consensus.Subscriber{regenSubscriber, agreementEventSubscriber}
}

// Finalize implements consensus.Component.
func (g *generator) Finalize() {}

func (g *generator) CollectRegeneration(e consensus.Event) error {
	g.generateBlock()
}

func (g *generator) CollectAgreementEvent(e consensus.Event) error {
	cert := g.generateCertificate(e)
	buf := new(bytes.Buffer)
	if err := block.MarshalCertificate(buf, cert); err != nil {
		return err
	}

	g.publisher.Publish(topics.Certificate, buf)
	return nil
}

func (g *generator) generateBlock() {
	blk, sev, err := g.blockGen.Generate(roundUpdate)
	if err != nil {
		log.WithFields(log.Fields{
			"process": "generation",
		}).WithError(err).Warnln("error generating proof and block")
		return
	}

	marshalledEvent := b.marshalScore(sev)
	marshalledBlock := b.marshalBlock(blk)
	g.publisher.Publish(topics.Gossip, marshalledEvent)
	g.publisher.Publish(topics.Gossip, marshalledBlock)
}

func (g *generator) generateCertificate() *block.Certificate {
	return &block.Certificate{
		StepOneBatchedSig: a.agreementEvent.VotesPerStep[0].Signature.Compress(),
		StepTwoBatchedSig: a.agreementEvent.VotesPerStep[1].Signature.Compress(),
		Step:              a.agreementEvent.Header.Step,
		StepOneCommittee:  a.agreementEvent.VotesPerStep[0].BitSet,
		StepTwoCommittee:  a.agreementEvent.VotesPerStep[1].BitSet,
	}
}

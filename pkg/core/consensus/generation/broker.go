package generation

import (
	"bytes"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

// FIXME: this component is in a VERY sorry state
func newComponent(publisher eventbus.Publisher, rpcBus *rpcbus.RPCBus) *generator {
	return &generator{
		publisher: publisher,
		rpcBus:    rpcBus,
	}
}

type generator struct {
	roundInfo consensus.RoundUpdate
	k         ristretto.Scalar
	publisher eventbus.Publisher
	rpcBus    *rpcbus.RPCBus

	blockGen BlockGenerator
}

// Initialize the generator, by creating listeners for the desired topics.
// Implements consensus.Component.
func (g *generator) Initialize(signer consensus.Signer, ru consensus.RoundUpdate) []consensus.Subscriber {
	g.roundInfo = ru

	// If we are not in this round's bid list, we can skip initialization, as there
	// would be no need to listen for these events if we are not qualified to generate
	// scores and blocks.
	//if !g.blockGen.proofGenerator.InBidList(g.bidList) {
	//	return nil
	//}

	regenSubscriber := consensus.Subscriber{
		Topic:    topics.Regeneration,
		Listener: consensus.NewSimpleListener(g.CollectRegeneration),
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
	return nil
}

func (g *generator) CollectAgreementEvent(e consensus.Event) error {
	a := agreement.Agreement{}
	if err := agreement.Unmarshal(&e.Payload, &a); err != nil {
		return err
	}

	cert := g.generateCertificate(a)

	buf := new(bytes.Buffer)
	if err := block.MarshalCertificate(buf, cert); err != nil {
		return err
	}

	g.publisher.Publish(topics.Certificate, buf)
	return nil
}

func (g *generator) generateBlock() {
	blk, sev, err := g.blockGen.Generate(g.roundInfo)
	if err != nil {
		log.WithFields(log.Fields{
			"process": "generation",
		}).WithError(err).Warnln("error generating proof and block")
		return
	}

	scoreBuf := new(bytes.Buffer)
	if err := selection.MarshalScoreEvent(scoreBuf, &sev); err != nil {
		// TODO: log this
		return
	}

	blockBuf := new(bytes.Buffer)
	if err := block.Marshal(blockBuf, &blk); err != nil {
		// TODO: log this
		return
	}

	g.publisher.Publish(topics.Gossip, scoreBuf)
	g.publisher.Publish(topics.Gossip, blockBuf)
}

func (g *generator) generateCertificate(a agreement.Agreement) *block.Certificate {
	return &block.Certificate{
		StepOneBatchedSig: a.VotesPerStep[0].Signature.Compress(),
		StepTwoBatchedSig: a.VotesPerStep[1].Signature.Compress(),
		Step:              a.Header.Step,
		StepOneCommittee:  a.VotesPerStep[0].BitSet,
		StepTwoCommittee:  a.VotesPerStep[1].BitSet,
	}
}

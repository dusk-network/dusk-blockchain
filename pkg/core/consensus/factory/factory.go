package factory

import (
	"bytes"
	"encoding/binary"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/notary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
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
	eventBus       *wire.EventBus
	initSubscriber *wire.EventSubscriber
	initChannel    chan uint64

	timerLength time.Duration
	committee   committee.Committee
}

// New returns an initialized ConsensusFactory.
func New(eventBus *wire.EventBus, timerLength time.Duration,
	committee committee.Committee) *ConsensusFactory {

	initChannel := make(chan uint64, 1)

	initCollector := &initCollector{initChannel}
	initSubscriber := wire.NewEventSubscriber(eventBus,
		initCollector, msg.InitializationTopic)

	return &ConsensusFactory{
		eventBus:       eventBus,
		initSubscriber: initSubscriber,
		initChannel:    initChannel,
		timerLength:    timerLength,
		committee:      committee,
	}
}

// StartConsensus will start listening to the initialization topic, wait for
// a message to come in, and then proceed to start the consensus components.
func (c *ConsensusFactory) StartConsensus() {
	go c.initSubscriber.Accept()

	round := <-c.initChannel

	// start processes
	scoreSelector := selection.NewScoreSelector(c.eventBus, c.timerLength,
		msg.VerifyEd25519Signature, zkproof.Verify)

	scoreSelector.Listen()

	setFilter := selection.NewSigSetFilter(c.eventBus, msg.VerifyEd25519Signature,
		c.committee, c.timerLength)

	setFilter.Listen()

	blockReducer := reduction.NewBlockReducer(c.eventBus, msg.VerifyEd25519Signature,
		c.committee, c.timerLength)

	blockReducer.Listen()

	setReducer := reduction.NewSigSetReducer(c.eventBus, msg.VerifyEd25519Signature,
		c.committee, c.timerLength)

	setReducer.Listen()

	blockNotary := notary.NewBlockNotary(c.eventBus, msg.VerifyEd25519Signature,
		c.committee)

	blockNotary.Listen()

	setNotary := notary.NewSigSetNotary(c.eventBus, msg.VerifyEd25519Signature,
		c.committee, round)

	setNotary.Listen()
}

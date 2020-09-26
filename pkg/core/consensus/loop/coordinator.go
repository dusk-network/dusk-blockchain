package loop

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation/score"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// Consensus is the state machine that runs the steps of consensus. Rather than
// trying to coordinate the various steps, it lets them execute and indicate
// which should be the following status, until completion.
// Each round is run separately and in a synchronous, blocking fashion (except
// the Agreement which should be run asynchronously by design)
type Consensus struct {
	*consensus.Emitter
	eventQueue *consensus.Queue
	roundQueue *consensus.Queue

	agreementChan chan message.Message
	eventChan     chan message.Message
	score *consensus.Phase
}

// New creates a new Consensus struct. The legacy StopConsensus and RoundUpdate
// are now replaced with context cancelation and direct function call operated
// by the chain component
func New(emitter *consensus.Emitter) *Consensus {
	// TODO: channel size should be configurable
	agreementChan := make(chan message.Message, 1000)
	eventChan := make(chan message.Message, 1000)

	// subscribe agreement phase to message.Agreement
	aChan := eventbus.NewChanListener(agreementChan)
	emitter.EventBus.Subscribe(topics.Agreement, aChan)

	// subscribe topics to eventChan
	evSub := eventbus.NewChanListener(eventChan)
	emitter.EventBus.AddDefaultTopics(topics.Reduction, topics.Score)
	emitter.EventBus.SubscribeDefault(evSub)

	// TODO: instantiate all components/phases (eventually get inspired by the consensus
	// factory)
	redu2 := secondstep.New()
	redu1 := firststep.New(redu2)
	sel := selection.New(red1)
	scr := score.New(sel)

	c := &Consensus{
		Emitter:       emitter,
		eventQueue:    consensus.NewQueue(),
		roundQueue:    consensus.NewQueue(),
		agreementChan: agreementChan,
		eventChan:     eventChan,
		score:         scr,
	}

	return c
}

// Spin the consensus state machine. The consensus runs for the whole round
// until either a new round is produced or the node needs to re-sync. The
// Agreement loop (acting roundwise) runs concurrently with the generation-selection-reduction
// loop (acting step-wise)
func (c *Consensus) Spin(ctx context.Context, round consensus.RoundUpdate) error {
	// cancel
	ctx, cancel := context.WithCancel(ctx)

	// errors thrown by the Agreement
	errChan := make(chan error, 1)

	// the agreement loop needs to be running until either the consensus
	// reaches a maximum amount of iterations (approx. 213 steps), or we get
	// agreements from future rounds and stopped receiving them for the current round
	// (in which case we should probably re-sync)
	go func() {
		err := agreement.Run(ctx, c.roundQueue, c.agreementChan, round, c.Emitter)
		// canceling the consensus phase loop when Agreement is done (either
		// because the parent canceled or because a consensus has been reached)
		cancel()
		errChan <- err
	}()

	// score generation phase is the first step in the consensus
	phase := c.score.Run
	// synchronous consensus loop keeps running until the agreement invokes
	// context.Done or the context is canceled some other way
	for step := uint8(1); phase != nil; step++ {
		phase = phase(ctx, c.eventQueue, c.eventChan, round, step)
	}

	// if we are here, either:
	// - agreement completed normally and we can move on to the next block
	// - agreement completed with an error and a re-synchronization
	// - caller canceled the context (we are likely re-synchronizing with
	// the network)
	// - we reached the maximum amount of steps (~213) and the consensus should
	// halt. In this case, cancel() will take care of stopping the Agreement
	// loop

	return <-errChan
}

//phase should start by
// - cleaning the events from the previous round
// - cleaning the events from the previous steps
// - pulling from EventQueue the events it is interested in (current
// step/category... this is a bit weird, why do we need both?)

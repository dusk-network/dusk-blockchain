package consensus

import (
	"bytes"
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// PhaseFn represents the recursive consensus state function
type PhaseFn func(context.Context, *Queue, chan message.Message, uint64, uint8, *Emitter) PhaseFn

// Consensus is the state machine that runs the steps of consensus. Rather than
// trying to coordinate the various steps, it lets them execute and indicate
// which should be the following status, until completion.
// Each round is run separately and in a synchronous, blocking fashion (except
// the Agreement which should be run asynchronously by design)
type Consensus struct {
	*Emitter
	eventQueue *Queue
	roundQueue *Queue

	agreementChan chan message.Agreement
	eventChan     chan message.Message
}

// Emitter is a simple struct to pass the communication channels that the steps should be
// able to emit onto
type Emitter struct {
	eventBus    *eventbus.EventBus
	rpcBus      *rpcbus.RPCBus
	keys        key.Keys
	pubkeyBuf   bytes.Buffer
	proxy       transactions.Proxy
	pubKey      *transactions.PublicKey
	timerLength time.Duration
}

func Start(emitter *Emitter) *Consensus {
	return &Consensus{
		Emitter:    emitter,
		eventQueue: NewQueue(),
		roundQueue: NewQueue(),
	}
}

// Run the consensus state machine
func (c *Consensus) Run(ctx context.Context, round uint64) {
	ctx := context.Background()

	// the agreement loop needs to be running until either the consensus
	// reaches a maximum amount of iterations (approx. 213 steps), or we get
	// agreements from future rounds and stopped receiving them for the current round
	// (in which case we should probably re-sync)
	go agreement.Run(ctx, c.roundQueue, agreementChan, round, c.emitter)

	// generation phase is the first step in the consensus
	phase := generation.Run
	// synchronous consensus loop keeps running until the agreement invokes
	// context.Done or the context is canceled some other way
	for step := uint8(1); phase != nil; step++ {
		phase = phase(ctx, c.eventQueue, eventChan, round, step, c.emitter)
	}

	// if we are here, either:
	// - agreement completed and we can move on to the next block
	// - agreement canceled the context and triggered a re-synchronization
	// - caller canceled the context (we are likely re-synchronizing with
	// the network)
	// - we reached the maximum amount of steps (~213) and the consensus should
	// halt
}

//phase should start by
// - cleaning the events from the previous round
// - cleaning the events from the previous steps
// - pulling from EventQueue the events it is interested in (current
// step/category... this is a bit weird, why do we need both?)

package reduction

import (
	"bytes"
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func launchAccumulator(ctx *context, publisher wire.EventPublisher, eventChan <-chan wire.Event) *accumulator {
	accumulator := newAccumulator(ctx, publisher, eventChan)
	go accumulator.accumulate()
	return accumulator
}

type accumulator struct {
	sea                *consensus.StepEventAccumulator
	publisher          wire.EventPublisher
	ctx                *context
	collectedVotesChan chan []wire.Event
	eventChan          <-chan wire.Event
	reducer            *reducer
}

func newAccumulator(ctx *context, publisher wire.EventPublisher, eventChan <-chan wire.Event) *accumulator {
	return &accumulator{
		sea:                consensus.NewStepEventAccumulator(),
		publisher:          publisher,
		ctx:                ctx,
		collectedVotesChan: make(chan []wire.Event, 1),
		eventChan:          eventChan,
	}
}

func (a *accumulator) accumulate() {
	for {
		ev := <-a.eventChan
		if err := a.ctx.handler.Verify(ev); err != nil {
			log.WithFields(log.Fields{
				"process": "reduction",
				"error":   err,
			}).Warnln("verification failed")
			continue
		}

		a.repropagate(ev)
		b := new(bytes.Buffer)
		if err := a.ctx.handler.EmbedVoteHash(ev, b); err == nil {
			hash := hex.EncodeToString(b.Bytes())
			count := a.sea.Store(ev, hash)
			if count > a.ctx.committee.Quorum() {
				votes := a.sea.Map[hash]
				a.collectedVotesChan <- votes
				a.sea.Clear()
			}
		}
	}
}

func (a *accumulator) repropagate(ev wire.Event) {
	buf := new(bytes.Buffer)
	_ = a.ctx.handler.Marshal(buf, ev)
	message, _ := wire.AddTopic(buf, topics.BlockReduction)
	a.publisher.Publish(string(topics.Gossip), message)
}

// There is no mutex involved here, as this function is only ever called by the broker,
// who does it synchronously. The reducer variable therefore can not be in a race
// condition with another goroutine.
func (a *accumulator) startReduction(hash []byte) {
	log.Traceln("Starting Reduction")
	a.forwardSelection(hash)
	a.reducer = newReducer(a.collectedVotesChan, a.ctx, a.publisher)
	go a.reducer.begin(a.sea)
}

func (a *accumulator) forwardSelection(hash []byte) {
	buf := bytes.NewBuffer(hash)
	vote, err := a.ctx.handler.MarshalHeader(buf, a.ctx.state)
	if err != nil {
		panic(err)
	}

	a.publisher.Publish(msg.OutgoingBlockReductionTopic, vote)
}

func (a *accumulator) updateRound(round uint64) {
	log.WithFields(log.Fields{
		"process": "reducer",
		"round":   round,
	}).Debugln("Updating round")
	if a.reducer != nil {
		a.reducer.stale = true
		a.reducer.end()
		a.reducer = nil
	}
	a.ctx.state.Update(round)
	a.sea.Clear()
}

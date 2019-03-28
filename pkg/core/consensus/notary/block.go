package notary

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// LaunchBlockNotary creates a BlockNotary by injecting the EventBus, the
// CommitteeStore and the message validation primitive. The blocknotary is
// returned solely for test purposes
func LaunchBlockNotary(eventBus *wire.EventBus,
	c committee.Committee) *blockNotary {

	blockCollector := initBlockCollector(eventBus, c)
	selectionChan := selection.InitSigSetSelectionCollector(eventBus)

	blockNotary := &blockNotary{
		eventBus:        eventBus,
		roundUpdateChan: consensus.InitRoundUpdate(eventBus),
		blockCollector:  blockCollector,
		selectionChan:   selectionChan,
	}

	go blockNotary.Listen()
	return blockNotary
}

type (
	// BlockEvent expresses a vote on a block hash. It is a real type alias of
	// committee.Event
	BlockEvent = committee.NotaryEvent

	// BlockNotary notifies when there is a consensus on a block hash
	blockNotary struct {
		eventBus        *wire.EventBus
		blockCollector  *blockCollector
		roundUpdateChan chan uint64
		selectionChan   chan bool
	}

	// BlockEventUnmarshaller is the unmarshaller of BlockEvents. It is a real
	// type alias of notaryEventUnmarshaller
	BlockEventUnmarshaller = committee.EventUnMarshaller

	// blockCollector collects CommitteeEvent. When a Quorum is reached, it
	// propagates the new Block Hash to the proper channel
	blockCollector struct {
		*committee.Collector
		BlockChan    chan []byte
		Result       *BlockEvent
		Unmarshaller *committee.NotaryEventUnMarshaller
	}
)

// Listen to block agreement messages and signature set agreement messages and
// propagates round and phase updates.
// A round update should be propagated when we get enough sigSetAgreement messages
// for a given step
// A phase update should be propagated when we get enough blockAgreement messages
// for a certain blockhash
// BlockNotary gets a currentRound somehow
func (b *blockNotary) Listen() {
	for {
		select {
		case blockHash := <-b.blockCollector.BlockChan:
			b.sendResult()
			buffer := bytes.NewBuffer(blockHash)
			b.eventBus.Publish(msg.PhaseUpdateTopic, buffer)
		case round := <-b.roundUpdateChan:
			b.blockCollector.Result = nil
			b.blockCollector.UpdateRound(round)
		case <-b.selectionChan:
			b.sendResult()
		}
	}
}

func (b *blockNotary) sendResult() error {
	if b.blockCollector.Result == nil {
		return nil
	}

	buffer := new(bytes.Buffer)
	topicBytes := topics.TopicToByteArray(topics.SigSet)
	if _, err := buffer.Write(topicBytes[:]); err != nil {
		return err
	}

	if err := b.blockCollector.Unmarshaller.Marshal(buffer, b.blockCollector.Result); err != nil {
		return err
	}

	b.eventBus.Publish(string(topics.SigSet), buffer)
	return nil
}

// newBlockCollector is injected with the committee, a channel where to publish
// the new Block Hash and the validator function for shallow checking of the
// marshalled form of the CommitteeEvent messages.
func newBlockCollector(c committee.Committee) *blockCollector {
	cc := &committee.Collector{
		StepEventCollector: make(map[string][]wire.Event),
		Committee:          c,
	}

	return &blockCollector{
		Collector: cc,
		BlockChan: make(chan []byte, 1),
		Unmarshaller: committee.NewNotaryEventUnMarshaller(committee.NewReductionEventUnMarshaller(nil),
			msg.VerifyEd25519Signature),
	}
}

// InitBlockCollector is a helper to minimize the wiring of EventSubscribers,
// collector and channels
func initBlockCollector(eventBus *wire.EventBus, c committee.Committee) *blockCollector {
	collector := newBlockCollector(c)
	go wire.NewEventSubscriber(eventBus, collector,
		string(msg.BlockAgreementTopic)).Accept()
	return collector
}

// Collect as specifiec in the EventCollector interface. It dispatches the
// unmarshalled CommitteeEvent to Process method
func (c *blockCollector) Collect(buffer *bytes.Buffer) error {
	ev := committee.NewNotaryEvent() // BlockEvent is an alias of committee.Event
	if err := c.Unmarshaller.Unmarshal(buffer, ev); err != nil {
		return err
	}

	isIrrelevant := c.CurrentRound != 0 && c.CurrentRound != ev.Round
	if c.ShouldBeSkipped(ev) || isIrrelevant {
		return nil
	}

	c.Process(ev)
	return nil
}

// Process checks if the quorum is reached and if it isn't simply stores the Event
// in the proper step
func (c *blockCollector) Process(event *BlockEvent) {
	nrAgreements := c.Store(event, string(event.Step))
	// did we reach the quorum?
	if nrAgreements >= c.Committee.Quorum() {
		// notify the Notary
		go func() { c.BlockChan <- event.BlockHash }()
		c.Result = event
		c.Clear()
	}
}

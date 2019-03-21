package notary

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// BlockEvent expresses a vote on a block hash. It is a real type alias of notaryEvent
type BlockEvent = committee.Event

// BlockEventUnmarshaller is the unmarshaller of BlockEvents. It is a real type alias of notaryEventUnmarshaller
type BlockEventUnmarshaller = committee.EventUnMarshaller

// BlockCollector collects CommitteeEvent. When a Quorum is reached, it propagates the new Block Hash to the proper channel
type BlockCollector struct {
	*committee.Collector
	BlockChan    chan []byte
	Unmarshaller wire.EventUnmarshaller
}

// NewBlockCollector is injected with the committee, a channel where to publish the new Block Hash and the validator function for shallow checking of the marshalled form of the CommitteeEvent messages.
func NewBlockCollector(c committee.Committee, validateFunc func(*bytes.Buffer) error) *BlockCollector {

	cc := &committee.Collector{
		StepEventCollector: make(map[uint8][]wire.Event),
		Committee:          c,
	}

	return &BlockCollector{
		Collector:    cc,
		BlockChan:    make(chan []byte, 1),
		Unmarshaller: committee.NewEventUnMarshaller(validateFunc),
	}
}

// InitBlockCollector is a helper to minimize the wiring of EventSubscribers, collector and channels
func InitBlockCollector(eventBus *wire.EventBus, c committee.Committee, validateFunc func(*bytes.Buffer) error) *BlockCollector {
	collector := NewBlockCollector(c, validateFunc)
	go wire.NewEventSubscriber(eventBus, collector,
		string(msg.BlockAgreementTopic)).Accept()
	return collector
}

// Collect as specifiec in the EventCollector interface. It dispatches the unmarshalled CommitteeEvent to Process method
func (c *BlockCollector) Collect(buffer *bytes.Buffer) error {
	ev := committee.NewEvent() // BlockEvent is an alias of committee.Event
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

// Process checks if the quorum is reached and if it isn't simply stores the Event in the proper step
func (c *BlockCollector) Process(event *BlockEvent) {
	nrAgreements := c.Store(event, event.Step)
	// did we reach the quorum?
	if nrAgreements >= c.Committee.Quorum() {
		// notify the Notary
		go func() { c.BlockChan <- event.BlockHash }()
		c.Clear()
	}
}

// BlockNotary notifies when there is a consensus on a block hash
type BlockNotary struct {
	eventBus       *wire.EventBus
	blockCollector *BlockCollector
	roundCollector *consensus.RoundCollector
}

// NewBlockNotary creates a BlockNotary by injecting the EventBus, the CommitteeStore and the message validation primitive
func NewBlockNotary(eventBus *wire.EventBus,
	validateFunc func(*bytes.Buffer) error,
	c committee.Committee) *BlockNotary {

	roundCollector := consensus.InitRoundCollector(eventBus)
	blockCollector := InitBlockCollector(eventBus, c, validateFunc)

	return &BlockNotary{
		eventBus:       eventBus,
		roundCollector: roundCollector,
		blockCollector: blockCollector,
	}
}

// Listen to block agreement messages and signature set agreement messages and propagates round and phase updates.
// A round update should be propagated when we get enough sigSetAgreement messages for a given step
// A phase update should be propagated when we get enough blockAgreement messages for a certain blockhash
// BlockNotary gets a currentRound somehow
func (b *BlockNotary) Listen() {
	for {
		select {
		case blockHash := <-b.blockCollector.BlockChan:
			buffer := bytes.NewBuffer(blockHash)
			b.eventBus.Publish(msg.PhaseUpdateTopic, buffer)
		case round := <-b.roundCollector.RoundChan:
			b.blockCollector.UpdateRound(round)
		}
	}
}

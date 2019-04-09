package notary

import (
	"bytes"
	"encoding/binary"

	log "github.com/sirupsen/logrus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// LaunchBlockNotary creates a BlockNotary by injecting the EventBus, the
// CommitteeStore and the message validation primitive. The blocknotary is
// returned solely for test purposes
func LaunchBlockNotary(eventBus *wire.EventBus, c committee.Committee,
	currentRound uint64) *blockNotary {

	blockCollector := initBlockCollector(eventBus, c, currentRound)

	blockNotary := &blockNotary{
		eventBus:       eventBus,
		blockCollector: blockCollector,

		// TODO: review
		repropagationChannel: blockCollector.RepropagationChannel,
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
		eventBus       *wire.EventBus
		blockCollector *blockCollector

		// TODO: review
		repropagationChannel chan *bytes.Buffer
		generationChan       chan bool
	}

	// blockCollector collects CommitteeEvent. When a Quorum is reached, it
	// propagates the new Block Hash to the proper channel
	blockCollector struct {
		*committee.Collector
		BlockChan    chan []byte
		RoundChan    chan uint64
		Unmarshaller wire.EventUnMarshaller
		queue        map[uint64][]*BlockEvent
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
		case round := <-b.blockCollector.RoundChan:
			// Marshalling the round update
			bs := make([]byte, 8)
			binary.LittleEndian.PutUint64(bs, round)
			buf := bytes.NewBuffer(bs)
			// publishing to the EventBus
			b.eventBus.Publish(msg.RoundUpdateTopic, buf)
		// case blockHash := <-b.blockCollector.BlockChan:
		case ev := <-b.repropagationChannel:
			// TODO: review
			message, _ := wire.AddTopic(ev, topics.BlockAgreement)
			b.eventBus.Publish(string(topics.Gossip), message)
		}
	}
}

// newBlockCollector is injected with the committee, a channel where to publish
// the new Block Hash and the validator function for shallow checking of the
// marshalled form of the CommitteeEvent messages.
func newBlockCollector(c committee.Committee, currentRound uint64) *blockCollector {
	cc := &committee.Collector{
		StepEventCollector:   consensus.NewStepEventCollector(),
		Committee:            c,
		CurrentRound:         currentRound,
		RepropagationChannel: make(chan *bytes.Buffer, 100),
	}

	return &blockCollector{
		Collector:    cc,
		BlockChan:    make(chan []byte, 1),
		RoundChan:    make(chan uint64, 1),
		Unmarshaller: committee.NewNotaryEventUnMarshaller(msg.VerifyEd25519Signature),
	}
}

// InitBlockCollector is a helper to minimize the wiring of EventSubscribers,
// collector and channels
func initBlockCollector(eventBus *wire.EventBus, c committee.Committee,
	currentRound uint64) *blockCollector {

	collector := newBlockCollector(c, currentRound)
	go wire.NewEventSubscriber(eventBus, collector,
		string(topics.BlockAgreement)).Accept()
	collector.RoundChan <- currentRound
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

	if c.shouldBeStored(ev) {
		c.queue[ev.Round] = append(c.queue[ev.Round], ev)
	}

	// TODO: review
	c.repropagate(ev)
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
		// c.BlockChan <- event.AgreedHash
		c.Clear()
		c.nextRound()

		log.WithField("process", "notary").Traceln("block agreement reached")
	}
}

func (c *blockCollector) nextRound() {
	log.WithField("process", "notary").Traceln("updating round")
	c.UpdateRound(c.CurrentRound + 1)
	// notify the Notary
	c.RoundChan <- c.CurrentRound
	c.Clear()

	// picking messages related to next round (now current)
	currentEvents := c.queue[c.CurrentRound]
	// processing messages store so far
	for _, event := range currentEvents {
		c.Process(event)
	}
}

func (c *blockCollector) shouldBeStored(event *BlockEvent) bool {
	return event.Round > c.CurrentRound
}

// TODO: review
func (c *blockCollector) repropagate(event *BlockEvent) {
	buf := new(bytes.Buffer)
	c.Unmarshaller.Marshal(buf, event)
	c.RepropagationChannel <- buf
}

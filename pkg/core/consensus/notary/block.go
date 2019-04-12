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

type (
	// BlockEvent expresses a vote on a block hash. It is a real type alias of
	// committee.Event
	BlockEvent = committee.NotaryEvent

	agreementMarshaller interface {
		wire.EventUnMarshaller
		wire.SignatureMarshaller
	}

	// blockCollector collects CommitteeEvent. When a Quorum is reached, it
	// propagates the new Block Hash to the proper channel
	blockCollector struct {
		publisher wire.EventPublisher
		*committee.Collector
		Unmarshaller agreementMarshaller
		queue        map[uint64][]*BlockEvent
	}
)

// LaunchBlockAgreement is a helper to minimize the wiring of TopicListeners,
// collector and channels
func LaunchBlockAgreement(eventBus *wire.EventBus, c committee.Committee,
	currentRound uint64) *blockCollector {
	collector := newBlockCollector(eventBus, c, currentRound)
	collector.updateRound(currentRound)
	go wire.NewTopicListener(eventBus, collector,
		string(topics.BlockAgreement)).Accept()
	return collector
}

// newBlockCollector is injected with the committee, a channel where to publish
// the new Block Hash and the validator function for shallow checking of the
// marshalled form of the CommitteeEvent messages.
func newBlockCollector(publisher wire.EventPublisher, c committee.Committee, currentRound uint64) *blockCollector {
	cc := &committee.Collector{
		StepEventAccumulator: consensus.NewStepEventAccumulator(),
		Committee:            c,
		CurrentRound:         currentRound,
	}

	return &blockCollector{
		Collector:    cc,
		publisher:    publisher,
		Unmarshaller: committee.NewNotaryEventUnMarshaller(msg.VerifyEd25519Signature),
	}
}

// Collect as specifiec in the EventCollector interface. It dispatches the
// unmarshalled CommitteeEvent to Process method
func (c *blockCollector) Collect(buffer *bytes.Buffer) error {
	ev := committee.NewNotaryEvent() // BlockEvent is an alias of committee.Event
	if err := c.Unmarshaller.Unmarshal(buffer, ev); err != nil {
		log.WithFields(log.Fields{
			"process": "notary",
			"error":   err,
		}).Warnln("error unmarshalling event")
		return err
	}

	if c.ShouldBeSkipped(ev) {
		log.WithFields(log.Fields{
			"process": "notary",
			"error":   "sender not a committee member",
		}).Debugln("event dropped")
		return nil
	}

	c.repropagate(ev)
	isIrrelevant := c.CurrentRound > ev.Round && c.CurrentRound != 0
	if isIrrelevant {
		log.WithFields(log.Fields{
			"process":         "notary",
			"collector round": c.CurrentRound,
			"message round":   ev.Round,
			"message step":    ev.Step,
		}).Debugln("event irrelevant")
		return nil
	}

	if c.shouldBeStored(ev) {
		log.WithFields(log.Fields{
			"process":         "notary",
			"collector round": c.CurrentRound,
			"message round":   ev.Round,
			"message step":    ev.Step,
		}).Debugln("event stored")
		c.queue[ev.Round] = append(c.queue[ev.Round], ev)
	}

	// TODO: review
	c.Process(ev)
	return nil
}

// Process checks if the quorum is reached and if it isn't simply stores the Event
// in the proper step
func (c *blockCollector) Process(event *BlockEvent) {
	nrAgreements := c.Store(event, string(event.Step))
	log.WithFields(log.Fields{
		"process": "notary",
		"step":    event.Step,
		"count":   nrAgreements,
	}).Debugln("collected agreement event")
	// did we reach the quorum?
	if nrAgreements >= c.Committee.Quorum() {
		log.WithField("process", "notary").Traceln("quorum reached")
		// notify the Notary
		c.Clear()
		c.updateRound(c.CurrentRound + 1)

		log.WithField("process", "notary").Traceln("block agreement reached")
	}
}

func (c *blockCollector) updateRound(round uint64) {
	log.WithFields(log.Fields{
		"process": "notary",
		"round":   round,
	}).Debugln("updating round")

	c.UpdateRound(round)
	// Marshalling the round update
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, round)
	buf := bytes.NewBuffer(bs)
	// publishing to the EventBus
	c.publisher.Publish(msg.RoundUpdateTopic, buf)
	c.Clear()

	// picking messages related to next round (now current)
	currentEvents := c.queue[round]
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
	c.Unmarshaller.MarshalEdFields(buf, event.EventHeader)
	c.Unmarshaller.Marshal(buf, event)
	message, _ := wire.AddTopic(buf, topics.BlockAgreement)
	c.publisher.Publish(string(topics.Gossip), message)
}

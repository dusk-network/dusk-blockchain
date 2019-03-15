package agreement

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type AgreementEventCollector struct {
	StepEventCollector
	committee    user.Committee
	currentRound uint64
	resultChan   chan<- *CommitteeEvent
}

func NewAgreementEventCollector(committee user.Committee, resultChan chan *CommitteeEvent) *AgreementEventCollector {
	return &AgreementEventCollector{
		currentRound: 0,
		resultChan:   resultChan,
		committee:    committee,
	}
}

func (c *AgreementEventCollector) Collect(event *CommitteeEvent) error {
	if !c.shouldBeSkipped(event) {
		nrAgreements := c.Store(event)
		// did we reach the quorum?
		if nrAgreements >= c.committee.Quorum() {
			// notify the Notary
			go func() { c.resultChan <- event }()
			c.Clear()
		}
	}
	return nil
}

func (c *AgreementEventCollector) UpdateRound(round uint64) {
	c.currentRound = round
}

// ShouldBeSkipped checks if the message is not propagated by a committee member, that is not a duplicate (and in this case should probably check if the Provisioner is malicious) and that is relevant to the current round
// NOTE: currentRound is handled by some other process, so it is not this component's responsibility to handle corner cases (for example being on an obsolete round because of a disconnect, etc)
func (c *AgreementEventCollector) shouldBeSkipped(m *CommitteeEvent) bool {
	isDupe := c.Contains(m)
	isPleb := !m.BelongsToCommittee(c.committee)
	isIrrelevant := c.currentRound != m.Round
	err := c.verify(m)
	failedVerification := err != nil
	return isDupe || isPleb || isIrrelevant || failedVerification
}

func (c *AgreementEventCollector) verify(m *CommitteeEvent) error {
	return c.committee.VerifyVoteSet(m.VoteSet, m.BlockHash, m.Round, m.Step)
}

// BlockNotary notifies when there is a consensus on a block hash
type BlockNotary struct {
	eventBus               *wire.EventBus
	agreementMsgSubscriber *EventCommitteeSubscriber
	collector              *AgreementEventCollector
	roundMsgSubscriber     *EventSubscriber
	aggrChan               <-chan *CommitteeEvent
	roundChan              chan *bytes.Buffer
}

// NewBlockNotary creates a BlockNotary by injecting the EventBus, the CommitteeStore and the message validation primitive
func NewBlockNotary(eventBus *wire.EventBus,
	validateFunc func(*bytes.Buffer) error,
	committee *user.CommitteeStore) *BlockNotary {

	aggrChan := make(chan *CommitteeEvent, 1)
	roundChan := make(chan *bytes.Buffer, 1)

	collector := NewAgreementEventCollector(committee, aggrChan)
	roundSubscriber := NewEventSubscriber(eventBus, msg.RoundUpdateTopic)
	subscriber := NewEventCommitteeSubscriber(eventBus,
		newAgreementEventDecoder(validateFunc),
		collector, string(msg.BlockAgreementTopic))

	return &BlockNotary{
		eventBus:               eventBus,
		agreementMsgSubscriber: subscriber,
		roundMsgSubscriber:     roundSubscriber,
		aggrChan:               aggrChan,
		roundChan:              roundChan,
		collector:              collector,
	}
}

// Listen to block agreement messages and signature set agreement messages and propagates round and phase updates.
// A round update should be propagated when we get enough sigSetAgreement messages for a given step
// A phase update should be propagated when we get enough blockAgreement messages for a certain blockhash
// BlockNotary gets a currentRound somehow
func (b *BlockNotary) Listen() {
	go b.agreementMsgSubscriber.ReceiveEventCommittee()
	go b.roundMsgSubscriber.ReceiveEvent(b.roundChan)
	for {
		select {
		case message := <-b.aggrChan:
			buffer := bytes.NewBuffer(message.BlockHash)
			b.eventBus.Publish(msg.PhaseUpdateTopic, buffer)
		case roundUpdate := <-b.roundChan:
			round := binary.LittleEndian.Uint64(roundUpdate.Bytes())
			b.collector.UpdateRound(round)
		}
	}
}

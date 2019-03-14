package agreement

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type AgreementEventCollector struct {
	StepEventCollector
	committee    user.Committee
	currentRound uint64
	resultChan   chan<- *AgreementMessage
}

func NewAgreementEventCollector(committee user.Committee, resultChan chan *AgreementMessage) *AgreementEventCollector {
	return &AgreementEventCollector{
		currentRound: 0,
		resultChan:   resultChan,
		committee:    committee,
	}
}

func (c *AgreementEventCollector) Contains(event CommitteeEvent) bool {
	agreementMsg, ok := event.(*AgreementMessage)
	if !ok {
		return false
	}

	return c.IsDuplicate(agreementMsg, agreementMsg.Step)
}

func (c *AgreementEventCollector) Aggregate(event CommitteeEvent) error {
	agreementMsg := event.(*AgreementMessage)
	if !c.shouldBeSkipped(agreementMsg) {
		nrAgreements := c.Store(agreementMsg, agreementMsg.Step)
		// did we reach the quorum?
		if nrAgreements >= c.committee.Quorum() {
			// notify the Notary
			go func() { c.resultChan <- agreementMsg }()
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
func (c *AgreementEventCollector) shouldBeSkipped(m *AgreementMessage) bool {
	isDupe := c.Contains(m)
	isPleb := !m.BelongsToCommittee(c.committee)
	isIrrelevant := c.currentRound != m.Round
	err := c.verify(m)
	failedVerification := err != nil
	return isDupe || isPleb || isIrrelevant || failedVerification
}

func (c *AgreementEventCollector) verify(m *AgreementMessage) error {
	return c.committee.VerifyVoteSet(m.VoteSet, m.BlockHash, m.Round, m.Step)
}

// BlockNotary notifies when there is a consensus on a block hash
type BlockNotary struct {
	*EventCommitteeSubscriber
	currentRound uint64
	aggrChan     <-chan *AgreementMessage
}

// NewBlockNotary creates a BlockNotary by injecting the EventBus, the CommitteeStore and the message validation primitive
func NewBlockNotary(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error, committee *user.CommitteeStore) *BlockNotary {
	aggrChan := make(chan *AgreementMessage, 1)
	collector := NewAgreementEventCollector(committee, aggrChan)
	subscriber := NewEventCommitteeSubscriber(eventBus, newDecoder(validateFunc), collector, string(msg.BlockAgreementTopic))

	return &BlockNotary{
		EventCommitteeSubscriber: subscriber,
		currentRound:             0,
		aggrChan:                 aggrChan,
	}
}

// Listen to block agreement messages and signature set agreement messages and propagates round and phase updates.
// A round update should be propagated when we get enough sigSetAgreement messages for a given step
// A phase update should be propagated when we get enough blockAgreement messages for a certain blockhash
// BlockNotary gets a currentRound somehow
func (b *BlockNotary) Listen() {
	go b.Receive(msg.BlockAgreementTopic)
	for {
		select {
		case message := <-b.aggrChan:
			buffer := bytes.NewBuffer(message.BlockHash)
			b.eventBus.Publish(msg.PhaseUpdateTopic, buffer)
		}
	}
}

package agreement

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// BlockNotary notifies when there is a consensus on a block hash
type BlockNotary struct {
	eventBus              *wire.EventBus
	blockAgreementChannel <-chan *bytes.Buffer
	blockAgreementID      uint32
	quitChannel           <-chan *bytes.Buffer
	quitID                uint32
	committeeStore        *user.CommitteeStore
	currentRound          uint64
	decoder               EventDecoder
	agreementCollector    *StepEventCollector
}

// NewBlockNotary creates a BlockNotary by injecting the EventBus, the CommitteeStore and the message validation primitive
func NewBlockNotary(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error,
	committeeStore *user.CommitteeStore) *BlockNotary {
	quitChannel := make(chan *bytes.Buffer, 1)
	blockAgreementChannel := make(chan *bytes.Buffer, 100)

	return &BlockNotary{
		eventBus:              eventBus,
		blockAgreementChannel: blockAgreementChannel,
		quitChannel:           quitChannel,
		committeeStore:        committeeStore,
		agreementCollector:    newStepEventCollector(),
		decoder:               newDecoder(validateFunc),
	}
}

// Listen to block agreement messages and signature set agreement messages and propagates round and phase updates.
// A round update should be propagated when we get enough sigSetAgreement messages for a given step
// A phase update should be propagated when we get enough blockAgreement messages for a certain blockhash
// BlockNotary gets a currentRound somehow
func (b *BlockNotary) Listen() {
	for {
		select {
		case <-b.quitChannel:
			b.eventBus.Unsubscribe(string(msg.BlockAgreementTopic), b.blockAgreementID)
			b.eventBus.Unsubscribe(string(msg.QuitTopic), b.quitID)
			return
		case blockAgreement := <-b.blockAgreementChannel:
			d, err := b.decoder.Decode(*blockAgreement)
			if err != nil {
				break
			}
			// casting to an AgreementMessage as we know what Decoder we are using
			am := d.(*AgreementMessage)
			if b.shouldBeSkipped(am) {
				break
			}

			err = b.Aggregate(d, am.Round, am.Step)
			if err != nil {
				break
			}
		}
	}
}

func (b *BlockNotary) Aggregate(event CommitteeEvent, round uint64, step uint8) error {

	// storing the event always since we are gonna cleanup anyway if quorum is reached
	nrAgreements := b.agreementCollector.Store(event, step)
	// did we reach the quorum?
	if nrAgreements >= b.committeeStore.Quorum() {
		// notify everybody
		b.notifyAgreedBlockHash(event.(*AgreementMessage).BlockHash)
		b.agreementCollector = newStepEventCollector()
	}

	return nil
}

// shouldBeSkipped checks if the message is not propagated by a committee member, that is not a duplicate (and in this case should probably check if the Provisioner is malicious) and that is relevant to the current round
// NOTE: currentRound is handled by some other process, so it is not this component's responsibility to handle corner cases (for example being on an obsolete round because of a disconnect, etc)
func (b *BlockNotary) shouldBeSkipped(m *AgreementMessage) bool {
	isDupe := b.agreementCollector.IsDuplicate(m, m.Step)
	isPleb := !m.BelongsToCommittee(b.committeeStore)
	isIrrelevant := b.currentRound != m.Round
	err := b.Verify(m)
	failedVerification := err != nil
	return isDupe || isPleb || isIrrelevant || failedVerification
}

func (b *BlockNotary) Verify(m *AgreementMessage) error {
	return b.committeeStore.VerifyVoteSet(m.VoteSet, m.BlockHash, m.Round, m.Step)
}

func (b *BlockNotary) notifyAgreedBlockHash(blockHash []byte) {
	buffer := bytes.NewBuffer(blockHash)
	b.eventBus.Publish(msg.PhaseUpdateTopic, buffer)
}

package agreement

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// BlockNotary notifies when there is a consensus on a block hash
type BlockNotary struct {
	eventBus               *wire.EventBus
	blockAgreementChannel  <-chan *bytes.Buffer
	blockAgreementID       uint32
	quitChannel            <-chan *bytes.Buffer
	quitID                 uint32
	committeeStore         *user.CommitteeStore
	currentRound           uint64
	blockAgreementsPerStep map[uint8][]*blockAgreementMessage
	validate               func(*bytes.Buffer) error
}

// NewBlockNotary creates a BlockNotary by injecting the EventBus, the CommitteeStore and the message validation primitive
func NewBlockNotary(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error,
	committeeStore *user.CommitteeStore) *BlockNotary {
	quitChannel := make(chan *bytes.Buffer, 1)
	blockAgreementChannel := make(chan *bytes.Buffer, 100)

	return &BlockNotary{
		eventBus:               eventBus,
		blockAgreementChannel:  blockAgreementChannel,
		quitChannel:            quitChannel,
		committeeStore:         committeeStore,
		blockAgreementsPerStep: make(map[uint8][]*blockAgreementMessage),
		validate:               validateFunc,
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
			b.eventBus.Unsubscribe(string("quit"), b.quitID)
			return
		case blockAgreement := <-b.blockAgreementChannel:
			if err := b.validate(blockAgreement); err != nil {
				return
			}

			blockAgreementMsg, err := decodeBlockAgreement(blockAgreement)
			if err != nil {
				break
			}
			b.processMsg(blockAgreementMsg)
		}
	}
}

// processMsg checks the message to see if it is valid, aka:
//	- run all validations on the message
//  - verify that the message is in the same round of the BlockNotary
func (b *BlockNotary) processMsg(m *blockAgreementMessage) {

	if b.shouldBeSkipped(m) {
		return
	}

	if b.shouldBeStored(m) {
		blockAgreementList := b.blockAgreementsPerStep[m.Step]
		if blockAgreementList == nil {
			blockAgreementList = make([]*blockAgreementMessage, b.committeeStore.Threshold)
		}

		// storing the agreement vote for the proper step
		blockAgreementList = append(blockAgreementList, m)
		b.blockAgreementsPerStep[m.Step] = blockAgreementList
		return
	}

	// if we reach the treshold for a step, this invalidates the agreements still pending
	//confirming the blockhash voted upon
	b.notifyAgreedBlockHash(m.BlockHash)
	// clean the map up
	b.blockAgreementsPerStep = make(map[uint8][]*blockAgreementMessage)
}

// shouldBeSkipped checks if the message is not propagated by a committee member, that is not a duplicate (and in this case should probably check if the Provisioner is malicious) and that is relevant to the current round
// NOTE: currentRound is handled by some other process, so it is not this component's responsibility to handle corner cases (for example being on an obsolete round because of a disconnect, etc)
func (b *BlockNotary) shouldBeSkipped(m *blockAgreementMessage) bool {
	isDupe := b.isDuplicate(m)
	isPleb := !b.committeeStore.PartakesInCommittee(m.PubKeyBLS)
	isIrrelevant := b.currentRound != m.Round
	// err := b.verifyVote(m.VoteSet, m.Step )
	// failedVerification := err != nil
	return isDupe || isPleb || isIrrelevant // || failedVerification
}

// shouldBeStored checks if the message did not trigger a threshold
func (b *BlockNotary) shouldBeStored(m *blockAgreementMessage) bool {
	agreementList := b.blockAgreementsPerStep[m.Step]
	return agreementList == nil || len(agreementList)+1 < b.committeeStore.Threshold
}

func (b *BlockNotary) isDuplicate(m *blockAgreementMessage) bool {
	for _, previousAgreement := range b.blockAgreementsPerStep[m.Step] {
		if m.equal(previousAgreement) {
			return true
		}
	}

	return false
}

func (b *BlockNotary) notifyAgreedBlockHash(blockHash []byte) {
	buffer := bytes.NewBuffer(blockHash)
	b.eventBus.Publish(msg.PhaseUpdateTopic, buffer)
}

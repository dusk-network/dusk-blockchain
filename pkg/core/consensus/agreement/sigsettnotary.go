package agreement

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// SigSetNotary notifies of the reached agreement (threshold) for a signature set. It listens to sigSetAgreement messages coming from the eventBus and notifies when the round should be updated (once signature threshold is reached for votes)
type SigSetNotary struct {
	eventBus                *wire.EventBus
	sigSetAgreementChannel  <-chan *bytes.Buffer
	sigSetAgreementID       uint32
	quitChannel             <-chan *bytes.Buffer
	quitID                  uint32
	committeeStore          *user.Committee
	currentRound            uint64
	sigSetAgreementPerSteps map[uint8][]*sigSetAgreementMessage
	validate                func(*bytes.Buffer) error
}

// NewSigSetNotary creates a new SigSetNotary. The message validation function, the eventBus and the committeeStore are injected
func NewSigSetNotary(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error, committeeStore *user.CommitteeStore) *SigSetNotary {
	quitChannel := make(chan *bytes.Buffer, 1)
	sigSetAgreementChannel := make(chan *bytes.Buffer, 100)

	return &SigSetNotary{
		eventBus:                eventBus,
		sigSetAgreementChannel:  sigSetAgreementChannel,
		quitChannel:             quitChannel,
		committeeStore:          committeeStore,
		sigSetAgreementPerSteps: make(map[uint8][]*sigSetAgreementMessage),
		validate:                validateFunc,
	}
}

// Listen to block agreement messages and signature set agreement messages and propagates round and phase updates.
// A round update should be propagated when we get enough sigSetAgreement messages for a given step
// A phase update should be propagated when we get enough blockAgreement messages for a certain blockhash
// SigSetNotary gets a currentRound somehow
func (s *SigSetNotary) Listen() {
	for {
		select {
		case <-s.quitChannel:
			s.eventBus.Unsubscribe(string(msg.SigSetAgreementTopic), s.sigSetAgreementID)
			s.eventBus.Unsubscribe(string(msg.QuitTopic), s.quitID)
			return
		case sigSetAgreement := <-s.sigSetAgreementChannel:
			sigSetAgreementMsg, err := decodeSigSetAgreement(sigSetAgreement)
			if err != nil {
				break
			}
			s.processMsg(sigSetAgreementMsg)
		}
	}
}

// processSigSetAgreement is a phase-agnostic agreement function. It stores the amount of votes it has gotten for each step, and once the threshold is reached, the function
// will send the resulting hash into outputChannel, and return
func (s *SigSetNotary) processMsg(m *sigSetAgreementMessage) {

	if s.shouldBeSkipped(m) {
		return
	}

	if s.shouldBeStored(m) {
		sigSetAgreements := s.sigSetAgreementPerSteps[m.Step]
		if sigSetAgreements == nil {
			sigSetAgreements = make([]*sigSetAgreementMessage, s.committeeStore.Threshold())
		}

		// storing the sigSetAgreement for the proper step
		sigSetAgreements = append(sigSetAgreements, m)
		s.sigSetAgreementPerSteps[m.Step] = sigSetAgreements
		return
	}

	s.currentRound = s.currentRound + 1
	//confirming the blockhash voted upon
	s.notifyRoundUpdate(s.currentRound)
	// clean the map up
	s.sigSetAgreementPerSteps = make(map[uint8][]*sigSetAgreementMessage)
}

func (s *SigSetNotary) shouldBeSkipped(m *sigSetAgreementMessage) bool {
	isDupe := s.isDuplicate(m)
	isPleb := !s.committeeStore.PartakesInCommittee(m.PubKeyBLS)
	isIrrelevant := s.currentRound != m.Round
	err := s.committeeStore.VerifyVoteSet(m.VoteSet, m.BlockHash, m.Round, m.Step)
	failedVerification := err != nil
	return isDupe || isPleb || isIrrelevant || failedVerification
}

func (s *SigSetNotary) shouldBeStored(m *sigSetAgreementMessage) bool {
	agreementList := s.sigSetAgreementPerSteps[m.Step]
	return agreementList == nil || m.Round > s.currentRound || len(agreementList)+1 < s.committeeStore.Threshold()
}

func (s *SigSetNotary) notifyRoundUpdate(r uint64) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, r)
	buf := bytes.NewBuffer(b)
	s.eventBus.Publish(msg.RoundUpdateTopic, buf)
}

// TODO: if we get duplicated messages with different data, we need to decrease the reputation of the provisioner
// TODO: can we use an interface for this
func (s *SigSetNotary) isDuplicate(m *sigSetAgreementMessage) bool {
	for _, previousAgreement := range s.sigSetAgreementPerSteps[m.Step] {
		if m.equal(previousAgreement) {
			return true
		}
	}

	return false
}

package collection

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
)

type SetSelector struct {
	eventBus           *wire.EventBus
	sigSetChannel      <-chan *bytes.Buffer
	sigSetID           uint32
	roundUpdateChannel <-chan *bytes.Buffer
	roundUpdateID      uint32
	quitChannel        <-chan *bytes.Buffer
	quitID             uint32

	round       uint64
	step        uint8
	totalWeight uint64
	timerLength time.Duration

	collecting       bool
	inputChannel     chan *sigSetMessage
	outputChannel    chan []byte
	setSizeThreshold int

	provisioners    *user.Provisioners
	votingCommittee map[string]uint8

	// injected functions
	validate func(*bytes.Buffer) error

	queue *sigSetQueue
}

// NewSetSelector will return a pointer to a ScoreSelector with the passed
// parameters.
func NewSetSelector(eventBus *wire.EventBus, timerLength time.Duration,
	validateFunc func(*bytes.Buffer) error) *SetSelector {

	queue := newSigSetQueue()
	sigSetChannel := make(chan *bytes.Buffer, 100)
	roundUpdateChannel := make(chan *bytes.Buffer, 1)
	quitChannel := make(chan *bytes.Buffer, 1)
	inputChannel := make(chan *sigSetMessage, 100)
	outputChannel := make(chan []byte, 1)

	setSelector := &SetSelector{
		eventBus:           eventBus,
		sigSetChannel:      sigSetChannel,
		roundUpdateChannel: roundUpdateChannel,
		quitChannel:        quitChannel,
		timerLength:        timerLength,
		validate:           validateFunc,
		inputChannel:       inputChannel,
		outputChannel:      outputChannel,
		queue:              &queue,
	}

	sigSetID := setSelector.eventBus.Subscribe(string(commands.SigSet), sigSetChannel)
	setSelector.sigSetID = sigSetID

	roundUpdateID := setSelector.eventBus.Subscribe(msg.RoundUpdateTopic, roundUpdateChannel)
	setSelector.roundUpdateID = roundUpdateID

	quitID := setSelector.eventBus.Subscribe("quit", quitChannel)
	setSelector.quitID = quitID

	return setSelector
}

// Listen will start the SetSelector up. It will decide when to run the
// collection logic, and manage the incoming messages with regards to the
// current consensus state.
func (s *SetSelector) Listen() {
	for {
		// Check queue first
		queuedMessages := s.queue.GetMessages(s.round, s.step)
		if queuedMessages != nil {
			for _, message := range queuedMessages {
				s.handleMessage(message)
			}
		}

		select {
		case <-s.quitChannel:
			s.eventBus.Unsubscribe(string(commands.SigSet), s.sigSetID)
			s.eventBus.Unsubscribe(msg.RoundUpdateTopic, s.roundUpdateID)
			s.eventBus.Unsubscribe("quit", s.quitID)
			return
		case result := <-s.outputChannel:
			s.collecting = false
			s.step++
			if err := s.setVotingCommittee(); err != nil {
				// Log
				return
			}

			buffer := bytes.NewBuffer(result)
			s.eventBus.Publish("outgoing", buffer)
		case <-s.roundUpdateChannel:
			s.moveToNextRound()
		case messageBytes := <-s.sigSetChannel:
			if err := s.validate(messageBytes); err != nil {
				break
			}

			message, err := decodeSigSetMessage(messageBytes)
			if err != nil {
				break
			}

			// If the ScoreCollector was just initialised, we start off from the
			// round and step of the first score message we receive.
			if s.round == 0 && s.step == 0 {
				s.round = message.Round
				s.step = message.Step
			}

			s.handleMessage(message)
		}
	}
}

func (s *SetSelector) handleMessage(message *sigSetMessage) {
	if s.shouldBeProcessed(message) {
		if !s.collecting {
			s.collecting = true
			go s.selectBestSignatureSet(s.inputChannel, s.outputChannel)
		}

		s.inputChannel <- message
	} else if s.shouldBeStored(message) {
		s.queue.PutMessage(message.Round, message.Step, message)
	}
}

// SelectBestSignatureSet will receive signature set candidate messages through
// setCandidateChannel for a set amount of time. The function will store the
// vote set of the node with the highest stake. When the timer runs out,
// the stored vote set is returned.
func (s *SetSelector) selectBestSignatureSet(inputChannel <-chan *sigSetMessage,
	outputChannel chan<- []byte) {

	// Variable to keep track of the highest stake we've received.
	var highest uint64

	// Variable to keep track of the vote set associated with the
	// highest stake we've received
	var bestVoteSet []*msg.Vote

	timer := time.NewTimer(s.timerLength)

	for {
		select {
		case <-timer.C:
			voteSetHash, err := hashVoteSet(bestVoteSet)
			if err != nil {
				// Log
				return
			}

			outputChannel <- voteSetHash
			return
		case m := <-inputChannel:
			err := s.verifySigSetMessage(m)
			if err != nil && err.Priority == prerror.High {
				// Log
				return
			} else if err != nil && err.Priority == prerror.Low {
				break
			}

			pubKeyStr := hex.EncodeToString(m.PubKeyBLS)
			if s.votingCommittee[pubKeyStr] != 0 {
				stake, err := s.provisioners.GetStake(m.PubKeyBLS)
				if err != nil {
					return
				}

				if stake > highest {
					highest = stake
					bestVoteSet = m.SigSet
				}
			}
		}
	}
}

func (s SetSelector) verifySigSetMessage(m *sigSetMessage) *prerror.PrError {
	// TODO: winning block hash check

	if err := s.verifyVoteSet(m.SigSet, m.WinningBlockHash, m.Step); err != nil {
		return err
	}

	return nil
}

func (s SetSelector) verifyVoteSet(voteSet []*msg.Vote, hash []byte,
	step uint8) *prerror.PrError {

	// Size check
	if err := s.validateVoteSetLength(voteSet); err != nil {
		return err
	}

	// Create committees
	committee1, err := s.provisioners.CreateVotingCommittee(s.round, s.totalWeight, step)
	if err != nil {
		return prerror.New(prerror.High, err)
	}

	committee2, err := s.provisioners.CreateVotingCommittee(s.round, s.totalWeight, step-1)
	if err != nil {
		return prerror.New(prerror.High, err)
	}

	for _, vote := range voteSet {
		if err := checkVoterEligibility(vote.PubKeyBLS, committee1, committee2); err != nil {
			return err
		}

		if err := verifyVote(vote, hash, step); err != nil {
			return err
		}
	}

	return nil
}

func (s SetSelector) validateVoteSetLength(voteSet []*msg.Vote) *prerror.PrError {
	if len(voteSet) < s.setSizeThreshold {
		return prerror.New(prerror.Low, errors.New("vote set is too small"))
	}

	return nil
}

func checkVoterEligibility(pubKeyBLS []byte, committee1,
	committee2 map[string]uint8) *prerror.PrError {

	pubKeyStr := hex.EncodeToString(pubKeyBLS)
	if committee1[pubKeyStr] == 0 {
		if committee2[pubKeyStr] == 0 {
			return prerror.New(prerror.Low, errors.New("voter is not eligible to vote"))
		}
	}

	return nil
}

func verifyVote(vote *msg.Vote, hash []byte, step uint8) *prerror.PrError {
	// A set should purely contain votes for one single hash
	if !bytes.Equal(hash, vote.VotedHash) {
		return prerror.New(prerror.Low, errors.New("voteset contains a vote "+
			"for the wrong hash"))
	}

	// A vote should be from the same step or the step before, in comparison
	// to the passed step parameter
	if notFromValidStep(vote.Step, step) {
		return prerror.New(prerror.Low, errors.New("vote is from another cycle"))
	}

	// Verify signature
	if err := verifyBLSSignature(vote.PubKeyBLS, vote.VotedHash, vote.SignedHash); err != nil {
		return prerror.New(prerror.Low, errors.New("BLS verification failed"))
	}

	return nil
}

func notFromValidStep(voteStep, setStep uint8) bool {
	return voteStep != setStep && voteStep+1 != setStep
}

func verifyBLSSignature(pubKeyBytes, message, signature []byte) error {
	pubKeyBLS := &bls.PublicKey{}
	if err := pubKeyBLS.Unmarshal(pubKeyBytes); err != nil {
		return err
	}

	sig := &bls.Signature{}
	if err := sig.Decompress(signature); err != nil {
		return err
	}

	apk := bls.NewApk(pubKeyBLS)
	return bls.Verify(apk, message, sig)
}

func hashVoteSet(voteSet []*msg.Vote) ([]byte, error) {
	// Encode signature set
	buffer := new(bytes.Buffer)
	for _, vote := range voteSet {
		if err := encodeVote(vote, buffer); err != nil {
			return nil, err
		}
	}

	// Hash bytes and set it on context
	sigSetHash, err := hash.Sha3256(buffer.Bytes())
	if err != nil {
		return nil, err
	}

	return sigSetHash, nil

}

func encodeVote(vote *msg.Vote, w io.Writer) error {
	if err := encoding.Write256(w, vote.VotedHash); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, vote.PubKeyBLS); err != nil {
		return err
	}

	if err := encoding.WriteBLS(w, vote.SignedHash); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, vote.Step); err != nil {
		return err
	}

	return nil
}

func (s *SetSelector) setVotingCommittee() error {
	votingCommittee, err := s.provisioners.CreateVotingCommittee(s.round,
		s.totalWeight, s.step)
	if err != nil {
		return err
	}

	s.votingCommittee = votingCommittee
	return nil
}

func (s *SetSelector) moveToNextRound() error {
	s.queue.Clear(s.round)
	// TODO: include round in roundupdate message buffer
	s.round++
	s.step = 1
	if err := s.setVotingCommittee(); err != nil {
		return err
	}

	return nil
}

func (s SetSelector) shouldBeProcessed(m *sigSetMessage) bool {
	correctRound := m.Round == s.round
	correctStep := m.Step == s.step

	pubKeyStr := hex.EncodeToString(m.PubKeyBLS)
	eligibleToVote := s.votingCommittee[pubKeyStr] > 0

	return correctRound && correctStep && eligibleToVote
}

func (s SetSelector) shouldBeStored(m *sigSetMessage) bool {
	return m.Round > s.round || m.Step > s.step
}

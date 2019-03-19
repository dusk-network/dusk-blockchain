package selection

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
)

// SetSelector contains information about the state of the consensus.
// It also maintains a message queue, with messages intended for the SetSelector.
type SetSelector struct {
	eventBus                *wire.EventBus
	sigSetChannel           <-chan *bytes.Buffer
	sigSetID                uint32
	roundUpdateChannel      <-chan *bytes.Buffer
	roundUpdateID           uint32
	winningBlockHashChannel <-chan *bytes.Buffer
	winningBlockHashID      uint32
	initializationChannel   <-chan *bytes.Buffer
	initializationID        uint32
	quitChannel             <-chan *bytes.Buffer
	quitID                  uint32

	round       uint64
	step        uint8
	timerLength time.Duration

	collecting       bool
	inputChannel     chan *sigSetMessage
	outputChannel    chan []byte
	winningBlockHash []byte

<<<<<<< Updated upstream
	committeeStore  *committee.Store
=======
	committee       user.Committee
>>>>>>> Stashed changes
	votingCommittee map[string]uint8

	// injected functions
	verifyEd25519Signature func(*bytes.Buffer) error
	verifyVoteSet          func([]*msg.Vote, []byte, uint64, uint8) *prerror.PrError

	queue *sigSetQueue
}

// NewSetSelector will return a pointer to a SetSelector with the passed
// parameters.
func NewSetSelector(eventBus *wire.EventBus, timerLength time.Duration,
	verifyEd25519SignatureFunc func(*bytes.Buffer) error,
<<<<<<< Updated upstream
	verifyVoteSetFunc func([]*msg.Vote, []byte, uint64, uint8) *prerror.PrError,
	committeeStore *committee.Store) *SetSelector {
=======
	committee user.Committee) *SetSelector {
>>>>>>> Stashed changes

	queue := newSigSetQueue()
	sigSetChannel := make(chan *bytes.Buffer, 100)
	roundUpdateChannel := make(chan *bytes.Buffer, 1)
	winningBlockHashChannel := make(chan *bytes.Buffer, 1)
	initializationChannel := make(chan *bytes.Buffer, 1)
	quitChannel := make(chan *bytes.Buffer, 1)
	inputChannel := make(chan *sigSetMessage, 100)
	outputChannel := make(chan []byte, 1)

	setSelector := &SetSelector{
		eventBus:                eventBus,
		sigSetChannel:           sigSetChannel,
		roundUpdateChannel:      roundUpdateChannel,
		winningBlockHashChannel: winningBlockHashChannel,
		initializationChannel:   initializationChannel,
		quitChannel:             quitChannel,
		timerLength:             timerLength,
		inputChannel:            inputChannel,
		outputChannel:           outputChannel,
		committee:               committee,
		verifyEd25519Signature:  verifyEd25519SignatureFunc,
		queue:                   &queue,
	}

	sigSetID := setSelector.eventBus.Subscribe(string(topics.SigSet), sigSetChannel)
	setSelector.sigSetID = sigSetID

	roundUpdateID := setSelector.eventBus.Subscribe(msg.RoundUpdateTopic, roundUpdateChannel)
	setSelector.roundUpdateID = roundUpdateID

	winningBlockHashID := setSelector.eventBus.Subscribe(string(topics.Agreement),
		winningBlockHashChannel)
	setSelector.winningBlockHashID = winningBlockHashID

	initializationID := setSelector.eventBus.Subscribe(msg.InitializationTopic,
		initializationChannel)
	setSelector.initializationID = initializationID

	quitID := setSelector.eventBus.Subscribe(msg.QuitTopic, quitChannel)
	setSelector.quitID = quitID

	return setSelector
}

// Listen will start the SetSelector up. It will decide when to run the
// collection logic, and manage the incoming messages with regards to the
// current consensus state.
func (s *SetSelector) Listen() {
	// First, wait to initialise
	if err := s.initialise(); err != nil {
		// Log
		return
	}

	for {
		select {
		case <-s.quitChannel:
			s.eventBus.Unsubscribe(string(topics.SigSet), s.sigSetID)
			s.eventBus.Unsubscribe(msg.RoundUpdateTopic, s.roundUpdateID)
			s.eventBus.Unsubscribe(string(topics.Agreement), s.winningBlockHashID)
			s.eventBus.Unsubscribe(msg.QuitTopic, s.quitID)
			return
		case result := <-s.outputChannel:
			s.collecting = false

			buffer := bytes.NewBuffer(result)
			s.eventBus.Publish(msg.SelectionResultTopic, buffer)

			s.step++
			if err := s.setVotingCommittee(); err != nil {
				// Log
				return
			}
		case roundBuffer := <-s.roundUpdateChannel:
			round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
			s.updateRound(round)
		case winningBlockHash := <-s.winningBlockHashChannel:
			s.winningBlockHash = winningBlockHash.Bytes()
		case messageBytes := <-s.sigSetChannel:
			if err := s.verifyEd25519Signature(messageBytes); err != nil {
				break
			}

			message, err := decodeSigSetMessage(messageBytes)
			if err != nil {
				break
			}

			s.handleMessage(message)
		default:
			if s.winningBlockHash != nil {
				queuedMessages := s.queue.GetMessages(s.round, s.step)
				if queuedMessages != nil {
					for _, message := range queuedMessages {
						s.handleMessage(message)
					}
				}
			}
		}
	}
}

func (s *SetSelector) initialise() error {
	roundBuffer := <-s.initializationChannel
	round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	s.round = round
	s.step = 1
	return s.setVotingCommittee()
}

func (s *SetSelector) handleMessage(message *sigSetMessage) {
	if s.shouldBeProcessed(message) && s.winningBlockHashKnown() {
		if !s.collecting {
			s.collecting = true
			go s.selectBestSignatureSet(s.inputChannel, s.outputChannel)
		}

		s.inputChannel <- message
	} else if s.shouldBeStored(message) || !s.winningBlockHashKnown() {
		s.queue.PutMessage(message.Round, message.Step, message)
	}
}

// selectBestSignatureSet will receive signature set candidate messages through
// inputChannel for a set amount of time. The function will store the
// vote set of the node with the highest stake. When the timer runs out,
// the stored vote set is returned.
func (s *SetSelector) selectBestSignatureSet(inputChannel <-chan *sigSetMessage,
	outputChannel chan<- []byte) {

	// Variable to keep track of the highest stake we've received.
	// var highest uint64

	// Variable to keep track of the vote set associated with the
	// highest stake we've received
	var bestVoteSet []*msg.Vote

	timer := time.NewTimer(s.timerLength)

	for {
		select {
		case <-timer.C:
			if bestVoteSet != nil {
				voteSetHash, err := msg.HashVoteSet(bestVoteSet)
				if err != nil {
					// Log
					return
				}

				outputChannel <- voteSetHash
			} else {
				outputChannel <- nil
			}

			return
		case m := <-inputChannel:
			prErr := s.verifySigSetMessage(m)
			if prErr != nil && prErr.Priority == prerror.High {
				// Log
				return
			} else if prErr != nil && prErr.Priority == prerror.Low {
				break
			}

			// committee, err := s.committee.GetVotingCommittee(s.round, s.step)
			// if err != nil {
			// 	return
			// }
			// stake, err := committee.GetStake(m.PubKeyBLS)
			// if err != nil {
			// 	return
			// }

			// if stake > highest {
			// 	highest = stake
			// 	bestVoteSet = m.SigSet
			// }
		}
	}
}

func (s SetSelector) verifySigSetMessage(m *sigSetMessage) *prerror.PrError {
	if !bytes.Equal(s.winningBlockHash, m.WinningBlockHash) {
		return prerror.New(prerror.Low, errors.New("vote set is for the wrong block hash"))
	}

	if err := verifyVoteSetSignature(m); err != nil {
		return err
	}

	if err := s.validateVoteSetLength(m.SigSet); err != nil {
		return err
	}

	if err := s.committee.VerifyVoteSet(m.SigSet, m.WinningBlockHash,
		m.Round, m.Step); err != nil {

		return err
	}

	return nil
}

func verifyVoteSetSignature(m *sigSetMessage) *prerror.PrError {
	voteSetHash, err := msg.HashVoteSet(m.SigSet)
	if err != nil {
		return prerror.New(prerror.High, err)
	}

	if err := msg.VerifyBLSSignature(m.PubKeyBLS, voteSetHash, m.SignedSigSet); err != nil {
		return prerror.New(prerror.Low, err)
	}

	return nil
}

func (s SetSelector) validateVoteSetLength(voteSet []*msg.Vote) *prerror.PrError {
<<<<<<< Updated upstream
	if len(voteSet) < s.committeeStore.Quorum() {
=======
	if len(voteSet) < s.committee.Quorum() {
>>>>>>> Stashed changes
		return prerror.New(prerror.Low, errors.New("vote set is too small"))
	}

	return nil
}

func (s *SetSelector) setVotingCommittee() error {
	// committee := s.committeeStore.Get()
	// votingCommittee, err := committee.CreateVotingCommittee(s.round,
	// 	s.committeeStore.TotalWeight, s.step)
	// if err != nil {
	// 	return err
	// }

	// s.votingCommittee = votingCommittee
	return nil
}

func (s *SetSelector) updateRound(round uint64) error {
	s.queue.Clear(s.round)
	s.winningBlockHash = nil
	s.round = round
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

func (s SetSelector) winningBlockHashKnown() bool {
	return s.winningBlockHash != nil
}

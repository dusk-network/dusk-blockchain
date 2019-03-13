package reduction

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
)

// Reducer contains information about the state of the consensus.
// It also maintains a message queue, with messages intended for the Reducer.
type Reducer struct {
	eventBus                *wire.EventBus
	blockReductionChannel   <-chan *bytes.Buffer
	blockReductionID        uint32
	sigSetReductionChannel  <-chan *bytes.Buffer
	sigSetReductionID       uint32
	roundUpdateChannel      <-chan *bytes.Buffer
	roundUpdateID           uint32
	winningBlockHashChannel <-chan *bytes.Buffer
	winningBlockHashID      uint32
	quitChannel             <-chan *bytes.Buffer
	quitID                  uint32

	round       uint64
	step        uint8
	timerLength time.Duration

	reducing         bool
	currentHash      []byte
	inSigSetPhase    bool
	inputChannel     chan reductionMessage
	outputChannel    chan []byte
	voteThreshold    int
	winningBlockHash []byte

	committeeStore  *user.CommitteeStore
	votingCommittee map[string]uint8

	// injected functions
	validate func(*bytes.Buffer) error

	queue *reductionQueue
}

// NewReducer will return a pointer to a Reducer with the passed
// parameters.
func NewReducer(eventBus *wire.EventBus, timerLength time.Duration,
	validateFunc func(*bytes.Buffer) error,
	committeeStore *user.CommitteeStore) *Reducer {

	queue := newReductionQueue()
	blockReductionChannel := make(chan *bytes.Buffer, 100)
	sigSetReductionChannel := make(chan *bytes.Buffer, 100)
	roundUpdateChannel := make(chan *bytes.Buffer, 1)
	winningBlockHashChannel := make(chan *bytes.Buffer, 1)
	quitChannel := make(chan *bytes.Buffer, 1)
	inputChannel := make(chan reductionMessage, 100)
	outputChannel := make(chan []byte, 1)

	reducer := &Reducer{
		eventBus:                eventBus,
		blockReductionChannel:   blockReductionChannel,
		sigSetReductionChannel:  sigSetReductionChannel,
		roundUpdateChannel:      roundUpdateChannel,
		winningBlockHashChannel: winningBlockHashChannel,
		quitChannel:             quitChannel,
		timerLength:             timerLength,
		inputChannel:            inputChannel,
		outputChannel:           outputChannel,
		committeeStore:          committeeStore,
		validate:                validateFunc,
		queue:                   &queue,
	}

	blockReductionID := reducer.eventBus.Subscribe(string(commands.BlockReduction),
		blockReductionChannel)
	reducer.blockReductionID = blockReductionID

	sigSetReductionID := reducer.eventBus.Subscribe(string(commands.SigSetReduction),
		sigSetReductionChannel)
	reducer.sigSetReductionID = sigSetReductionID

	roundUpdateID := reducer.eventBus.Subscribe(msg.RoundUpdateTopic,
		roundUpdateChannel)
	reducer.roundUpdateID = roundUpdateID

	winningBlockHashID := reducer.eventBus.Subscribe(string(commands.Agreement),
		winningBlockHashChannel)
	reducer.winningBlockHashID = winningBlockHashID

	quitID := reducer.eventBus.Subscribe(msg.QuitTopic, quitChannel)
	reducer.quitID = quitID

	return reducer
}

// Listen will start the Reducer up. It will decide when to run the
// reduction logic, and manage the incoming messages with regards to the
// current consensus state.
func (r *Reducer) Listen() {
	for {
		// Check queue first
		queuedMessages := r.queue.GetMessages(r.round, r.step)

		if queuedMessages != nil {
			for _, message := range queuedMessages {
				if message.IsSigSetReductionMessage() == r.inSigSetPhase {
					r.handleMessage(message)
				}
			}
		}

		select {
		case <-r.quitChannel:
			r.eventBus.Unsubscribe(string(commands.BlockReduction),
				r.blockReductionID)
			r.eventBus.Unsubscribe(string(commands.SigSetReduction),
				r.sigSetReductionID)
			r.eventBus.Unsubscribe(msg.RoundUpdateTopic, r.roundUpdateID)
			r.eventBus.Unsubscribe(string(commands.Agreement), r.winningBlockHashID)
			r.eventBus.Unsubscribe(msg.QuitTopic, r.quitID)
			return
		case result := <-r.outputChannel:
			r.eventBus.Publish("outgoing", bytes.NewBuffer(result))
		case roundBuffer := <-r.roundUpdateChannel:
			round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
			r.updateRound(round)
		case winningBlockHash := <-r.winningBlockHashChannel:
			r.moveToSigSetPhase(winningBlockHash.Bytes())
		case messageBytes := <-r.blockReductionChannel:
			if err := r.validate(messageBytes); err != nil {
				break
			}

			message, err := decodeBlockReductionMessage(messageBytes)
			if err != nil {
				break
			}

			if err := r.processMessage(message); err != nil {
				break
			}
		case messageBytes := <-r.sigSetReductionChannel:
			if err := r.validate(messageBytes); err != nil {
				break
			}

			message, err := decodeSigSetReductionMessage(messageBytes)
			if err != nil {
				break
			}

			if err := r.processMessage(message); err != nil {
				break
			}
		}
	}
}

func (r Reducer) processMessage(message reductionMessage) error {
	// If the Reducer was just initialised, we start off from the
	// round and step of the first reduction message we receive.
	if r.round == 0 && r.step == 0 {
		if err := r.initialise(message); err != nil {
			// Log
			return err
		}
	}

	r.handleMessage(message)
	return nil
}

// TODO: find a better way to do this
func (r *Reducer) initialise(message reductionMessage) error {
	commonFields := message.GetCommonFields()
	r.round = commonFields.Round
	r.step = commonFields.Step
	if err := r.setVotingCommittee(); err != nil {
		return err
	}

	r.updateVoteThreshold()
	if message.IsSigSetReductionMessage() {
		r.inSigSetPhase = true
	}

	return nil
}

func (r *Reducer) handleMessage(message reductionMessage) {
	if r.shouldBeProcessed(message) && r.isFromCorrectPhase(message) {
		if !r.reducing {
			r.reducing = true
			go r.reduce()
		}

		r.inputChannel <- message
	} else if r.shouldBeStored(message) || !r.isFromCorrectPhase(message) {
		commonFields := message.GetCommonFields()
		r.queue.PutMessage(commonFields.Round, commonFields.Step, message)
	}
}

// TODO: better name
// reduce will run a two-step reduction cycle. After two steps of reduction,
// the results are checked. If this check is successful, the resulting hash
// and the combination of the two vote sets is sent to the outputChannel.
func (r *Reducer) reduce() {
	// TODO: vote here

	hash1, voteSet1 := r.decideOnHash()
	r.incrementStep()

	// TODO: vote again here

	hash2, voteSet2 := r.decideOnHash()

	if r.reductionSuccessful(hash1, hash2, voteSet1, voteSet2) {
		fullVoteSet := append(voteSet1, voteSet2...)
		result, err := createReductionResult(hash2, fullVoteSet)
		if err != nil {
			// Log
			return
		}

		r.outputChannel <- result
	}

	r.reducing = false
	r.incrementStep()
}

func (r Reducer) reductionSuccessful(hash1, hash2 []byte, voteSet1,
	voteSet2 []*msg.Vote) bool {

	notNil := hash1 != nil && hash2 != nil
	sameResults := bytes.Equal(hash1, hash2)
	voteSetsAreValid := len(voteSet1) == r.voteThreshold && len(voteSet2) == r.voteThreshold

	return notNil && sameResults && voteSetsAreValid
}

func createReductionResult(hash []byte, voteSet []*msg.Vote) ([]byte, error) {
	buffer := new(bytes.Buffer)
	if err := encoding.Write256(buffer, hash); err != nil {
		return nil, err
	}

	encodedVoteSet, err := msg.EncodeVoteSet(voteSet)
	if err != nil {
		return nil, err
	}

	if _, err := buffer.Write(encodedVoteSet); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// decideOnHash is a phase-agnostic reduction function. It will
// receive reduction messages through the inputChannel for a set amount of time.
// The incoming messages will be stored in a VoteSet, and if one particular hash
// receives enough votes, the function returns that hash, along with the vote set.
func (r *Reducer) decideOnHash() ([]byte, []*msg.Vote) {
	// Keep track of how many votes have been cast for a specific hash
	votesPerHash := make(map[string]uint8)

	// Keep track of who has already voted in a step
	hasVoted := make(map[string]bool)

	// Create a vote set variable to store incoming votes in
	var voteSet []*msg.Vote

	timer := time.NewTimer(r.timerLength)

	for {
		select {
		case <-timer.C:
			return nil, nil
		case m := <-r.inputChannel:
			commonFields := m.GetCommonFields()
			pubKeyStr := hex.EncodeToString(commonFields.PubKeyBLS)
			if hasVoted[pubKeyStr] {
				break
			}

			hasVoted[pubKeyStr] = true
			if err := r.verifyReductionMessage(m); err != nil {
				break
			}

			votedHashStr := hex.EncodeToString(commonFields.VotedHash)

			// Add votes for this block
			votesPerHash[votedHashStr] += r.getNodeVoteCount(pubKeyStr)

			r.addVotesToVoteSet(voteSet, pubKeyStr, commonFields)

			if r.thresholdExceeded(votesPerHash[votedHashStr]) {
				// Clean up the vote set, to remove votes for other blocks.
				removeDeviatingVotes(voteSet, commonFields.VotedHash)

				return commonFields.VotedHash, voteSet
			}
		}
	}
}

func (r Reducer) verifyReductionMessage(m reductionMessage) error {
	commonFields := m.GetCommonFields()
	if err := msg.VerifyBLSSignature(commonFields.PubKeyBLS, commonFields.VotedHash,
		commonFields.SignedHash); err != nil {

		return err
	}

	if m.IsSigSetReductionMessage() {
		sigSetMessage := m.(*sigSetReductionMessage)
		if err := r.checkWinningBlockHash(sigSetMessage.WinningBlockHash); err != nil {
			return err
		}
	}

	return nil
}

func (r Reducer) checkWinningBlockHash(hash []byte) error {
	if !bytes.Equal(r.winningBlockHash, hash) {
		return errors.New("signature set reduction message contains the wrong " +
			"winning block hash")
	}

	return nil
}

func (r Reducer) addVotesToVoteSet(voteSet []*msg.Vote, pubKeyBLSStr string,
	commonFields reductionBase) {

	voteCount := r.getNodeVoteCount(pubKeyBLSStr)
	vote := createVote(commonFields)
	for i := uint8(0); i < voteCount; i++ {
		voteSet = append(voteSet, vote)
	}
}

func (r Reducer) getNodeVoteCount(pubKeyBLSStr string) uint8 {
	return r.votingCommittee[pubKeyBLSStr]
}

func createVote(commonFields reductionBase) *msg.Vote {
	return &msg.Vote{
		VotedHash:  commonFields.VotedHash,
		PubKeyBLS:  commonFields.PubKeyBLS,
		SignedHash: commonFields.SignedHash,
		Step:       commonFields.Step,
	}
}

func (r Reducer) thresholdExceeded(votes uint8) bool {
	return int(votes) >= r.voteThreshold
}

func removeDeviatingVotes(voteSet []*msg.Vote, hash []byte) {
	for i, vote := range voteSet {
		if !bytes.Equal(vote.VotedHash, hash) {
			voteSet = append(voteSet[:i], voteSet[i+1:]...)
		}
	}
}

func (r *Reducer) setVotingCommittee() error {
	committee := r.committeeStore.Get()
	votingCommittee, err := committee.CreateVotingCommittee(r.round,
		r.committeeStore.TotalWeight(), r.step)
	if err != nil {
		return err
	}

	r.votingCommittee = votingCommittee
	return nil
}

func (r *Reducer) updateVoteThreshold() {
	r.voteThreshold = int(float64(len(r.votingCommittee)) * 0.75)
}

func (r *Reducer) incrementStep() {
	r.step++
	if err := r.setVotingCommittee(); err != nil {
		// Log
		return
	}

	r.updateVoteThreshold()
}

func (r *Reducer) moveToSigSetPhase(winningBlockHash []byte) {
	r.winningBlockHash = winningBlockHash
	r.inSigSetPhase = true
	r.step = 1
}

func (r *Reducer) updateRound(round uint64) error {
	r.queue.Clear(r.round)
	r.winningBlockHash = nil
	r.inSigSetPhase = false
	r.round = round
	r.step = 1
	if err := r.setVotingCommittee(); err != nil {
		return err
	}

	r.updateVoteThreshold()
	return nil
}

func (r Reducer) shouldBeProcessed(m reductionMessage) bool {
	commonFields := m.GetCommonFields()
	correctRound := commonFields.Round == r.round
	correctStep := commonFields.Step == r.step

	pubKeyStr := hex.EncodeToString(commonFields.PubKeyBLS)
	eligibleToVote := r.votingCommittee[pubKeyStr] > 0

	return correctRound && correctStep && eligibleToVote
}

func (r Reducer) isFromCorrectPhase(m reductionMessage) bool {
	return r.inSigSetPhase == m.IsSigSetReductionMessage()
}

func (r Reducer) shouldBeStored(m reductionMessage) bool {
	commonFields := m.GetCommonFields()
	return commonFields.Round > r.round || commonFields.Step > r.step
}

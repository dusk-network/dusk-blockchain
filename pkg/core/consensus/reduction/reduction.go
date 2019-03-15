package reduction

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"

	"golang.org/x/crypto/ed25519"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

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
	selectionResultChannel  <-chan *bytes.Buffer
	selectionResultID       uint32
	winningBlockHashChannel <-chan *bytes.Buffer
	winningBlockHashID      uint32
	initializationChannel   <-chan *bytes.Buffer
	initializationID        uint32
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
	winningBlockHash []byte

	committeeStore  *user.CommitteeStore
	votingCommittee map[string]uint8
	*user.Keys

	// injected functions
	validate func(*bytes.Buffer) error

	queue *reductionQueue
}

// NewReducer will return a pointer to a Reducer with the passed
// parameters.
func NewReducer(eventBus *wire.EventBus, timerLength time.Duration,
	validateFunc func(*bytes.Buffer) error,
	committeeStore *user.CommitteeStore, keys *user.Keys) *Reducer {

	queue := newReductionQueue()
	blockReductionChannel := make(chan *bytes.Buffer, 100)
	sigSetReductionChannel := make(chan *bytes.Buffer, 100)
	roundUpdateChannel := make(chan *bytes.Buffer, 1)
	selectionResultChannel := make(chan *bytes.Buffer, 10)
	winningBlockHashChannel := make(chan *bytes.Buffer, 1)
	initializationChannel := make(chan *bytes.Buffer, 1)
	quitChannel := make(chan *bytes.Buffer, 1)
	inputChannel := make(chan reductionMessage, 100)
	outputChannel := make(chan []byte, 1)

	reducer := &Reducer{
		eventBus:                eventBus,
		blockReductionChannel:   blockReductionChannel,
		sigSetReductionChannel:  sigSetReductionChannel,
		roundUpdateChannel:      roundUpdateChannel,
		selectionResultChannel:  selectionResultChannel,
		winningBlockHashChannel: winningBlockHashChannel,
		initializationChannel:   initializationChannel,
		quitChannel:             quitChannel,
		timerLength:             timerLength,
		inputChannel:            inputChannel,
		outputChannel:           outputChannel,
		committeeStore:          committeeStore,
		Keys:                    keys,
		validate:                validateFunc,
		queue:                   &queue,
	}

	blockReductionID := reducer.eventBus.Subscribe(string(topics.BlockReduction),
		blockReductionChannel)
	reducer.blockReductionID = blockReductionID

	sigSetReductionID := reducer.eventBus.Subscribe(string(topics.SigSetReduction),
		sigSetReductionChannel)
	reducer.sigSetReductionID = sigSetReductionID

	roundUpdateID := reducer.eventBus.Subscribe(msg.RoundUpdateTopic,
		roundUpdateChannel)
	reducer.roundUpdateID = roundUpdateID

	selectionResultID := reducer.eventBus.Subscribe(msg.SelectionResultTopic,
		selectionResultChannel)
	reducer.selectionResultID = selectionResultID

	winningBlockHashID := reducer.eventBus.Subscribe(string(topics.Agreement),
		winningBlockHashChannel)
	reducer.winningBlockHashID = winningBlockHashID

	initializationID := reducer.eventBus.Subscribe(msg.InitializationTopic,
		initializationChannel)
	reducer.initializationID = initializationID

	quitID := reducer.eventBus.Subscribe(msg.QuitTopic, quitChannel)
	reducer.quitID = quitID

	return reducer
}

// Listen will start the Reducer up. It will decide when to run the
// reduction logic, and manage the incoming messages with regards to the
// current consensus state.
func (r *Reducer) Listen() {
	// First, wait to initialise
	if err := r.initialise(); err != nil {
		// Log
		return
	}

	for {
		select {
		case result := <-r.outputChannel:
			if r.eligibleToVote() && result != nil {
				if err := r.voteAgreement(result); err != nil {
					// Log
					return
				}
			}

			r.eventBus.Publish(msg.ReductionResultTopic, bytes.NewBuffer(r.currentHash))
			r.currentHash = nil
			r.incrementStep()
			r.reducing = false
		case selectionResult := <-r.selectionResultChannel:
			if r.currentHash == nil {
				r.currentHash = selectionResult.Bytes()

				if r.eligibleToVote() {
					r.voteReduction()
				}
			}
		case messageBytes := <-r.blockReductionChannel:
			if err := r.validate(messageBytes); err != nil {
				break
			}

			message, err := decodeBlockReductionMessage(messageBytes)
			if err != nil {
				break
			}

			r.handleMessage(message)
		case messageBytes := <-r.sigSetReductionChannel:
			if err := r.validate(messageBytes); err != nil {
				break
			}

			message, err := decodeSigSetReductionMessage(messageBytes)
			if err != nil {
				break
			}

			r.handleMessage(message)
		default:
			queuedMessages := r.queue.GetMessages(r.round, r.step)

			if queuedMessages != nil {
				for _, message := range queuedMessages {
					r.handleMessage(message)
				}
			}

			if !r.reducing {
				r.checkForUpdates()
			}
		}
	}
}

func (r *Reducer) initialise() error {
	roundBuffer := <-r.initializationChannel
	round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	r.round = round
	r.step = 1
	return r.setVotingCommittee()
}

// These are checks for events that change the Reducer's state. We should only
// check these channels while the Reducer is not currently running reduction.
// Additionally, the quitChannel check is in this function as well, so that
// we don't leave a reduction goroutine hanging if we get the quit signal.
func (r *Reducer) checkForUpdates() {
	select {
	case <-r.quitChannel:
		r.eventBus.Unsubscribe(string(topics.BlockReduction),
			r.blockReductionID)
		r.eventBus.Unsubscribe(string(topics.SigSetReduction),
			r.sigSetReductionID)
		r.eventBus.Unsubscribe(msg.RoundUpdateTopic, r.roundUpdateID)
		r.eventBus.Unsubscribe(msg.SelectionResultTopic, r.selectionResultID)
		r.eventBus.Unsubscribe(string(topics.Agreement), r.winningBlockHashID)
		r.eventBus.Unsubscribe(msg.QuitTopic, r.quitID)
		return
	case winningBlockHash := <-r.winningBlockHashChannel:
		r.moveToSigSetPhase(winningBlockHash.Bytes())
	case roundBuffer := <-r.roundUpdateChannel:
		round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
		r.updateRound(round)
	default:
		break
	}
}

func (r *Reducer) handleMessage(message reductionMessage) {
	if r.shouldBeProcessed(message) && r.isFromCorrectPhase(message) {
		if !r.reducing {
			r.reducing = true
			go r.runReduction()
		}

		r.inputChannel <- message
	} else if r.shouldBeStored(message) || !r.isFromCorrectPhase(message) {
		commonFields := message.GetCommonFields()
		r.queue.PutMessage(commonFields.Round, commonFields.Step, message)
	}
}

// runReduction will run a two-step reduction cycle. After two steps of voting,
// the results are put into a function that deals with the outcome.
func (r *Reducer) runReduction() {
	hash1, voteSet1 := r.decideOnHash()
	r.currentHash = hash1
	r.incrementStep()

	// If we're in the voting committee for this round, vote on our previously
	// received result
	if r.eligibleToVote() {
		if err := r.voteReduction(); err != nil {
			// Log
			return
		}
	}

	hash2, voteSet2 := r.decideOnHash()
	r.currentHash = hash2

	r.handleReductionResults(hash1, hash2, voteSet1, voteSet2)
}

// handleReductionResults will send the proper result to the output channel,
// depending on the passed parameters.
func (r Reducer) handleReductionResults(hash1, hash2 []byte, voteSet1,
	voteSet2 []*msg.Vote) {

	if r.reductionSuccessful(hash1, hash2, voteSet1, voteSet2) {
		fullVoteSet := append(voteSet1, voteSet2...)
		encodedVoteSet, err := msg.EncodeVoteSet(fullVoteSet)
		if err != nil {
			// Log
			return
		}

		r.outputChannel <- encodedVoteSet
	} else {
		r.outputChannel <- nil
	}
}

func (r Reducer) eligibleToVote() bool {
	pubKeyStr := hex.EncodeToString(r.BLSPubKey.Marshal())
	return r.votingCommittee[pubKeyStr] > 0
}

func (r Reducer) voteReduction() error {
	message, err := r.createReductionMessage()
	if err != nil {
		return err
	}

	signature := ed25519.Sign(*r.EdSecretKey, message.Bytes())
	fullMessage, err := r.addPubKeyAndSig(message, signature)
	if err != nil {
		return err
	}

	// Send to wire
	r.eventBus.Publish(msg.OutgoingReductionTopic, fullMessage)
	return nil
}

func (r Reducer) reductionSuccessful(hash1, hash2 []byte, voteSet1,
	voteSet2 []*msg.Vote) bool {

	notEmpty := !bytes.Equal(hash1, make([]byte, 32)) &&
		!bytes.Equal(hash2, make([]byte, 32))

	sameResults := bytes.Equal(hash1, hash2)

	voteSetsAreCorrectLength := len(voteSet1) >= r.committeeStore.Threshold() &&
		len(voteSet2) >= r.committeeStore.Threshold()

	return notEmpty && sameResults && voteSetsAreCorrectLength
}

func (r Reducer) voteAgreement(voteSetBytes []byte) error {
	message, err := r.createAgreementMessage(voteSetBytes)
	if err != nil {
		return err
	}

	signature := ed25519.Sign(*r.EdSecretKey, message.Bytes())
	fullMessage, err := r.addPubKeyAndSig(message, signature)
	if err != nil {
		return err
	}

	// Send to wire
	r.eventBus.Publish(msg.OutgoingAgreementTopic, fullMessage)
	return nil
}

func (r Reducer) addPubKeyAndSig(message *bytes.Buffer,
	signature []byte) (*bytes.Buffer, error) {

	buffer := bytes.NewBuffer([]byte(*r.EdPubKey))
	if err := encoding.Write512(buffer, signature); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(message.Bytes()); err != nil {
		return nil, err
	}

	return buffer, nil
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
			// Return empty hash and empty vote set if we did not get a winner
			// before timeout
			return make([]byte, 32), nil
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
			voteCount := r.votingCommittee[pubKeyStr]
			votesPerHash[votedHashStr] += voteCount

			// Add votes to vote set
			for i := uint8(0); i < voteCount; i++ {
				vote := createVote(commonFields)
				voteSet = append(voteSet, vote)
			}

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

func createVote(commonFields reductionBase) *msg.Vote {
	return &msg.Vote{
		VotedHash:  commonFields.VotedHash,
		PubKeyBLS:  commonFields.PubKeyBLS,
		SignedHash: commonFields.SignedHash,
		Step:       commonFields.Step,
	}
}

func (r Reducer) thresholdExceeded(votes uint8) bool {
	return int(votes) >= r.committeeStore.Threshold()
}

func removeDeviatingVotes(voteSet []*msg.Vote, hash []byte) {
	for i, vote := range voteSet {
		if !bytes.Equal(vote.VotedHash, hash) {
			voteSet = append(voteSet[:i], voteSet[i+1:]...)
		}
	}
}

func (r *Reducer) incrementStep() error {
	r.step++
	return r.setVotingCommittee()
}

func (r *Reducer) setVotingCommittee() error {
	committee := r.committeeStore.Get()
	votingCommittee, err := committee.CreateVotingCommittee(r.round,
		r.committeeStore.TotalWeight, r.step)
	if err != nil {
		return err
	}

	r.votingCommittee = votingCommittee
	return nil
}

func (r *Reducer) moveToSigSetPhase(winningBlockHash []byte) {
	r.winningBlockHash = winningBlockHash
	r.inSigSetPhase = true
	r.currentHash = nil
	r.step = 1
}

func (r *Reducer) updateRound(round uint64) error {
	r.queue.Clear(r.round)
	r.winningBlockHash = nil
	r.currentHash = nil
	r.inSigSetPhase = false
	r.round = round
	r.step = 1
	return r.setVotingCommittee()
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

func (r Reducer) createAgreementMessage(voteSetBytes []byte) (*bytes.Buffer, error) {
	buffer := bytes.NewBuffer(voteSetBytes)

	signedVoteSet, err := bls.Sign(r.BLSSecretKey, r.BLSPubKey, voteSetBytes)
	if err != nil {
		return nil, err
	}

	if err := encoding.WriteBLS(buffer, signedVoteSet.Compress()); err != nil {
		return nil, err
	}

	if err := encoding.WriteVarBytes(buffer, r.BLSPubKey.Marshal()); err != nil {
		return nil, err
	}

	if r.inSigSetPhase {
		if err := encoding.Write256(buffer, r.winningBlockHash); err != nil {
			return nil, err
		}
	} else {
		if err := encoding.Write256(buffer, r.currentHash); err != nil {
			return nil, err
		}
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, r.round); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buffer, r.step); err != nil {
		return nil, err
	}

	if r.inSigSetPhase {
		if err := encoding.Write256(buffer, r.currentHash); err != nil {
			return nil, err
		}
	}

	return buffer, nil
}

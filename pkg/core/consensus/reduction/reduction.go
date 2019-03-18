package reduction

import (
	"bytes"
	"encoding/binary"
	"time"

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
	case roundBuffer := <-r.roundUpdateChannel:
		round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	default:
		break
	}
}

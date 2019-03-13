package selection

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/big"
	"time"

	ristretto "github.com/bwesterb/go-ristretto"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
)

// ScoreSelector contains information about the state of the consensus.
// It also maintains a message queue, with messages intended for the ScoreSelector.
type ScoreSelector struct {
	eventBus           *wire.EventBus
	scoreChannel       <-chan *bytes.Buffer
	scoreID            uint32
	roundUpdateChannel <-chan *bytes.Buffer
	roundUpdateID      uint32
	quitChannel        <-chan *bytes.Buffer
	quitID             uint32

	round       uint64
	step        uint8
	timerLength time.Duration

	collecting    bool
	inputChannel  chan *scoreMessage
	outputChannel chan []byte

	// injected functions
	validate    func(*bytes.Buffer) error
	verifyProof func([]byte, []byte, []byte, []byte, []byte) bool

	bidList *user.BidList
	tau     uint64
	queue   *scoreQueue
}

// NewScoreSelector will return a pointer to a ScoreSelector with the passed
// parameters.
func NewScoreSelector(eventBus *wire.EventBus, timerLength time.Duration,
	validateFunc func(*bytes.Buffer) error,
	verifyProofFunc func([]byte, []byte, []byte, []byte, []byte) bool) *ScoreSelector {

	queue := newScoreQueue()
	bidList := &user.BidList{}
	scoreChannel := make(chan *bytes.Buffer, 100)
	roundUpdateChannel := make(chan *bytes.Buffer, 1)
	quitChannel := make(chan *bytes.Buffer, 1)
	inputChannel := make(chan *scoreMessage, 100)
	outputChannel := make(chan []byte, 1)

	scoreSelector := &ScoreSelector{
		eventBus:           eventBus,
		scoreChannel:       scoreChannel,
		roundUpdateChannel: roundUpdateChannel,
		quitChannel:        quitChannel,
		timerLength:        timerLength,
		bidList:            bidList,
		validate:           validateFunc,
		verifyProof:        verifyProofFunc,
		inputChannel:       inputChannel,
		outputChannel:      outputChannel,
		queue:              &queue,
	}

	scoreID := scoreSelector.eventBus.Subscribe(string(commands.Score), scoreChannel)
	scoreSelector.scoreID = scoreID

	roundUpdateID := scoreSelector.eventBus.Subscribe(msg.RoundUpdateTopic, roundUpdateChannel)
	scoreSelector.roundUpdateID = roundUpdateID

	quitID := scoreSelector.eventBus.Subscribe("quit", quitChannel)
	scoreSelector.quitID = quitID

	return scoreSelector
}

// Listen will start the ScoreSelector up. It will decide when to run the
// collection logic, and manage the incoming messages with regards to the
// current consensus state.
func (s *ScoreSelector) Listen() {
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
			s.eventBus.Unsubscribe(string(commands.Score), s.scoreID)
			s.eventBus.Unsubscribe(msg.RoundUpdateTopic, s.roundUpdateID)
			s.eventBus.Unsubscribe("quit", s.quitID)
			return
		case result := <-s.outputChannel:
			s.collecting = false

			buffer := bytes.NewBuffer(result)
			s.eventBus.Publish("outgoing", buffer)

			s.step++
		case roundBuffer := <-s.roundUpdateChannel:
			round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
			s.updateRound(round)
		case messageBytes := <-s.scoreChannel:
			if err := s.validate(messageBytes); err != nil {
				break
			}

			message, err := decodeScoreMessage(messageBytes)
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

func (s *ScoreSelector) handleMessage(message *scoreMessage) {
	if s.shouldBeProcessed(message) {
		if !s.collecting {
			s.collecting = true
			go s.selectBestBlockHash(s.inputChannel, s.outputChannel)
		}

		s.inputChannel <- message
	} else if s.shouldBeStored(message) {
		s.queue.PutMessage(message.Round, message.Step, message)
	}
}

// selectBestBlockHash will receive score messages from the inputChannel for a
// certain amount of time. It will store the highest score along with it's
// accompanied block hash, and will send the block hash associated to the highest
// score into the outputChannel once the timer runs out.
func (s *ScoreSelector) selectBestBlockHash(inputChannel <-chan *scoreMessage,
	outputChannel chan<- []byte) {

	// Variable to keep track of the highest score we've received.
	var highest uint64

	// Variable to keep track of the block hash associated with the
	// highest score we've received.
	var bestBlockHash []byte

	timer := time.NewTimer(s.timerLength)

	for {
		select {
		case <-timer.C:
			outputChannel <- bestBlockHash
			return
		case m := <-inputChannel:
			err := s.verifyScoreMessage(m)
			if err != nil && err.Priority == prerror.High {
				// Log
				return
			} else if err != nil && err.Priority == prerror.Low {
				break
			}

			// Reconstruct the score as an integer
			score := big.NewInt(0).SetBytes(m.Score).Uint64()
			if err := s.validateScore(score); err != nil {
				break
			}

			if score > highest {
				highest = score
				bestBlockHash = m.CandidateHash
			}
		}
	}
}

func (s ScoreSelector) validateScore(score uint64) error {
	if score < s.tau {
		return errors.New("candidate score below threshold")
	}

	return nil
}

func (s ScoreSelector) verifyScoreMessage(m *scoreMessage) *prerror.PrError {
	// Check first if the BidList contains valid bids
	if err := s.validateBidListSubset(m.BidListSubset); err != nil {
		return err
	}

	// Verify the proof
	seedScalar := ristretto.Scalar{}
	seedScalar.Derive(m.Seed)
	if !s.verifyProof(m.Proof, seedScalar.Bytes(), m.BidListSubset, m.Score, m.Z) {
		return prerror.New(prerror.Low, errors.New("proof verification failed"))
	}

	return nil
}

func (s ScoreSelector) validateBidListSubset(BidListSubsetBytes []byte) *prerror.PrError {
	BidListSubset, err := user.ReconstructBidListSubset(BidListSubsetBytes)
	if err != nil {
		return err
	}

	return s.bidList.ValidateBids(BidListSubset)
}

func (s *ScoreSelector) updateRound(round uint64) {
	s.queue.Clear(s.round)
	s.round = round
	s.step = 1
}

func (s ScoreSelector) shouldBeProcessed(m *scoreMessage) bool {
	return m.Round == s.round && m.Step == s.step
}

func (s ScoreSelector) shouldBeStored(m *scoreMessage) bool {
	return m.Round > s.round || m.Step > s.step
}

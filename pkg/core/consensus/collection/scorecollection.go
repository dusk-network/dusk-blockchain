package collection

import (
	"bytes"
	"errors"
	"math/big"
	"sync"
	"time"

	ristretto "github.com/bwesterb/go-ristretto"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"
)

const topic = "score"

// ScoreCollector contains information about the state of the consensus.
// It also maintains a message queue, with messages intended for the ScoreCollector.
type ScoreCollector struct {
	eventBus     *wire.EventBus
	scoreChannel <-chan *bytes.Buffer

	round       uint64
	step        uint8
	timerLength time.Duration

	collecting    bool
	inputChannel  chan *scoreMessage
	outputChannel chan []byte

	BidList *user.BidList
	tau     uint64
	queue   *user.Queue
}

// NewScoreCollector will return a pointer to a ScoreCollector with the passed
// parameters.
func NewScoreCollector(eventBus *wire.EventBus, scoreChannel chan *bytes.Buffer,
	round uint64, step uint8, timerLength time.Duration) *ScoreCollector {

	queue := &user.Queue{
		Map: new(sync.Map),
	}

	scoreCollector := &ScoreCollector{
		eventBus:     eventBus,
		scoreChannel: scoreChannel,
		round:        round,
		step:         step,
		timerLength:  timerLength,
		queue:        queue,
	}

	scoreCollector.eventBus.Register(topic, scoreChannel)
	return scoreCollector
}

// Listen will start the ScoreCollector up. It will decide when to run the
// collection logic, and manage the incoming messages with regards to the
// current consensus state.
func (s *ScoreCollector) Listen() {
	for {
		// TODO: Check queue first

		select {
		case messageBytes := <-s.scoreChannel:
			if err := msg.Validate(messageBytes); err != nil {
				break
			}

			message, err := decodeScoreMessage(messageBytes)
			if err != nil {
				break
			}

			if s.shouldBeProcessed(message) {
				if !s.collecting {
					s.collecting = true
					s.inputChannel = make(chan *scoreMessage, 100)
					s.outputChannel = make(chan []byte, 1)
					go s.selectBestBlockHash(s.inputChannel, s.outputChannel)
				}

				s.inputChannel <- message
			} else if s.shouldBeStored(message) {
				s.queue.Put(message.Round, message.Step, message)
			}
		case result := <-s.outputChannel:
			s.collecting = false

			// TODO: fill up the buffer with reduction message fields
			buffer := bytes.NewBuffer(result)
			s.eventBus.Publish("outgoing", buffer)
		}
	}
}

// selectBestBlockHash will receive score messages from the inputChannel for a
// certain amount of time. It will store the highest score along with it's
// accompanied block hash, and will send the best block hash into the outputChannel
// once the timer runs out.
func (s *ScoreCollector) selectBestBlockHash(inputChannel <-chan *scoreMessage,
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
			err := s.verifyProof(m)
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

func (s *ScoreCollector) validateScore(score uint64) error {
	if score < s.tau {
		return errors.New("candidate score below threshold")
	}

	return nil
}

func (s *ScoreCollector) verifyProof(m *scoreMessage) *prerror.PrError {
	// Check first if the BidList contains valid bids
	if err := s.validateBidListSubset(m.BidListSubset); err != nil {
		return err
	}

	// Verify the proof
	seedScalar := ristretto.Scalar{}
	seedScalar.Derive(m.Seed)
	if !zkproof.Verify(m.Proof, seedScalar.Bytes(), m.BidListSubset, m.Score, m.Z) {
		return prerror.New(prerror.Low, errors.New("proof verification failed"))
	}

	return nil
}

func (s *ScoreCollector) validateBidListSubset(BidListSubsetBytes []byte) *prerror.PrError {
	BidListSubset, err := user.ReconstructBidListSubset(BidListSubsetBytes)
	if err != nil {
		return err
	}

	return s.BidList.ValidateBids(BidListSubset)
}

func (s *ScoreCollector) shouldBeProcessed(m *scoreMessage) bool {
	return m.Round == s.round && m.Step == s.step
}

func (s *ScoreCollector) shouldBeStored(m *scoreMessage) bool {
	return m.Round > s.round || m.Step > s.step
}

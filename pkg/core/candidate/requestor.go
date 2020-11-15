package candidate

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	lg "github.com/sirupsen/logrus"
)

var log = lg.WithField("process", "candidate-requestor")

// Requestor serves to retrieve certain Candidate messages from peers in the
// network.
type Requestor struct {
	lock                 sync.RWMutex
	requestHash          []byte
	publisher            eventbus.Publisher
	requestCandidateChan <-chan rpcbus.Request
	candidateQueue       chan message.Candidate
}

// NewRequestor returns an initialized Requestor struct.
func NewRequestor(publisher eventbus.Publisher, rb *rpcbus.RPCBus) *Requestor {
	requestCandidateChan := make(chan rpcbus.Request, 1)
	if err := rb.Register(topics.GetCandidate, requestCandidateChan); err != nil {
		panic(err)
	}

	return &Requestor{
		requestHash:          make([]byte, 32),
		publisher:            publisher,
		requestCandidateChan: requestCandidateChan,
		candidateQueue:       make(chan message.Candidate, 100),
	}
}

// Listen allows the Requestor to receive incoming candidate requests and
// process them accordingly. Should be ran in a goroutine.
func (r *Requestor) Listen(ctx context.Context) {
	for {
		select {
		case m := <-r.requestCandidateChan:
			cm, err := r.requestCandidate(m)
			if err != nil {
				m.RespChan <- rpcbus.NewResponse(nil, err)
				continue
			}

			m.RespChan <- rpcbus.NewResponse(cm, nil)
		case <-ctx.Done():
			return
		}
	}
}

// ProcessCandidate will process a received Candidate message.
// Invalid and non-matching Candidate messages are discarded.
func (r *Requestor) ProcessCandidate(msg message.Message) ([]bytes.Buffer, error) {
	if err := Validate(msg); err != nil {
		return nil, err
	}

	cm := msg.Payload().(message.Candidate)
	r.lock.RLock()
	defer r.lock.RUnlock()
	if bytes.Equal(cm.Block.Header.Hash, r.requestHash) {
		r.candidateQueue <- cm
	}
	return nil, nil
}

func (r *Requestor) requestCandidate(m rpcbus.Request) (message.Candidate, error) {
	hash := m.Params.([]byte)
	r.setRequestHash(hash)
	defer r.setRequestHash(make([]byte, 32))

	if err := r.publishGetCandidate(hash); err != nil {
		return message.Candidate{}, nil
	}

	getCandidateTimeOut := config.Get().Timeout.TimeoutBrokerGetCandidate
	timer := time.NewTimer(time.Duration(getCandidateTimeOut) * time.Second)

	for {
		select {
		case <-timer.C:
			log.WithField("hash", hex.EncodeToString(hash)).Debug("failed to receive candidate from the network")
			return message.Candidate{}, errors.New("failed to receive candidate from the network")
		case cm := <-r.candidateQueue:
			return cm, nil
		}
	}
}

func (r *Requestor) publishGetCandidate(hash []byte) error {
	// Send a request for this specific candidate
	buf := bytes.NewBuffer(hash)
	// Ugh! Move encoding after the Gossip ffs
	if err := topics.Prepend(buf, topics.GetCandidate); err != nil {
		return err
	}
	msg := message.New(topics.GetCandidate, *buf)
	r.publisher.Publish(topics.Gossip, msg)
	return nil
}

func (r *Requestor) setRequestHash(hash []byte) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.requestHash = hash
}

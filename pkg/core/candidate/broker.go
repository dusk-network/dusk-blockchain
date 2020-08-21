package candidate

import (
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/republisher"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	lg "github.com/sirupsen/logrus"
)

var log = lg.WithField("process", "candidate-broker")

// Broker is the entry point for the candidate component. It manages
// an in-memory store of `Candidate` messages, and allows for the
// fetching of these messages through the `RPCBus`. It listens
// for incoming `Candidate` messages and puts them on the store.
// In case an internal component requests an absent `Candidate`
// message, the Broker can make a `GetCandidate` request to the rest
// of the network, and will attempt to provide the requesting component
// with it's needed `Candidate`.
type Broker struct {
	publisher   eventbus.Publisher
	republisher *republisher.Republisher
	*store
	lock sync.RWMutex

	republished map[string]struct{}

	acceptedBlockChan <-chan block.Block
	candidateChan     <-chan message.Candidate
	getCandidateChan  <-chan rpcbus.Request
}

// NewBroker returns an initialized Broker struct. It will still need
// to be started by calling `Listen`.
func NewBroker(broker eventbus.Broker, rpcBus *rpcbus.RPCBus) *Broker {
	acceptedBlockChan, _ := consensus.InitAcceptedBlockUpdate(broker)
	getCandidateChan := make(chan rpcbus.Request, 1)
	err := rpcBus.Register(topics.GetCandidate, getCandidateChan)
	if err != nil {
		log.WithError(err).Error("could not Register GetCandidate")
	}

	b := &Broker{
		publisher:         broker,
		store:             newStore(),
		republished:       make(map[string]struct{}),
		acceptedBlockChan: acceptedBlockChan,
		candidateChan:     initCandidateCollector(broker),
		getCandidateChan:  getCandidateChan,
	}

	b.republisher = republisher.New(broker, topics.Candidate, Validate, b.Validate)
	return b
}

// Listen for incoming `Candidate` messages, and internal requests.
// Should be run in a goroutine.
func (b *Broker) Listen() {
	for {
		select {
		case cm := <-b.candidateChan:
			b.storeCandidateMessage(cm)
		case r := <-b.getCandidateChan:
			// candidate requests from the RPCBus
			b.provideCandidate(r)
		case blk := <-b.acceptedBlockChan:
			// accepted blocks come from the consensus
			b.clearRepublishedHashes()
			b.Clear(blk.Header.Height)
		}
	}
}

// TODO: interface - rpcBus encoding will be removed
func (b *Broker) provideCandidate(r rpcbus.Request) {

	params := r.Params.(bytes.Buffer)
	cm := b.store.fetchCandidateMessage(params.Bytes())
	if cm.Block == nil {
		// If we don't have the candidate message, we should ask the network for it.
		var err error
		cm, err = b.requestCandidate(params.Bytes())
		if err != nil {
			r.RespChan <- rpcbus.Response{Resp: nil, Err: err}
			return
		}
	}

	r.RespChan <- rpcbus.Response{Resp: cm, Err: nil}
}

// ErrGetCandidateTimeout is an error specific to timeout happening on
// GetCandidate calls
var ErrGetCandidateTimeout = errors.New("request GetCandidate timeout")

// requestCandidate from peers around this node. The candidate can only be
// requested for 2 rounds (which provides some protection from keeping to
// request bulky stuff)
// TODO: encoding the category within the packet and specifying it as category
// is ugly af
func (b *Broker) requestCandidate(hash []byte) (message.Candidate, error) {
	// Send a request for this specific candidate
	buf := bytes.NewBuffer(hash)
	// Ugh! Move encoding after the Gossip ffs
	if err := topics.Prepend(buf, topics.GetCandidate); err != nil {
		return message.Candidate{}, err
	}

	msg := message.New(topics.GetCandidate, *buf)
	errList := b.publisher.Publish(topics.Gossip, msg)
	diagnostics.LogPublishErrors("candidate/broker.go, topics.Gossip, topics.GetCandidate", errList)

	//FIXME: Add option to configure timeout #614
	timer := time.NewTimer(2 * time.Second)
	for {
		select {
		case <-timer.C:
			return message.Candidate{}, ErrGetCandidateTimeout

		// We take control of `candidateChan`, to monitor incoming
		// candidates. There should be no race condition in reading from
		// the channel, as the only way this function can be called would
		// be through `Listen`. Any incoming candidates which don't match
		// our request will be passed down to the queue.
		case cm := <-b.candidateChan:
			if bytes.Equal(cm.Block.Header.Hash, hash) {
				return cm, nil
			}

			b.storeCandidateMessage(cm)
		}
	}
}

// Validate that the candidate block carried by a message has not been already republished.
// This function complies to the republisher.Validator interface
func (b *Broker) Validate(m message.Message) error {
	cm := m.Payload().(message.Candidate)
	b.lock.Lock()
	defer b.lock.Unlock()
	if _, ok := b.republished[string(cm.Block.Header.Hash)]; ok {
		return republisher.DuplicatePayloadError
	}

	b.republished[string(cm.Block.Header.Hash)] = struct{}{}
	return nil
}

func (b *Broker) clearRepublishedHashes() {
	b.lock.Lock()
	defer b.lock.Unlock()
	for k := range b.republished {
		delete(b.republished, k)
	}
}

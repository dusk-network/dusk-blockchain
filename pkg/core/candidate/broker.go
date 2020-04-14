package candidate

import (
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/republisher"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

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
	lock        sync.RWMutex
	queue       map[string]message.Candidate
	validHashes map[string]struct{}
	republished map[string]struct{}

	acceptedBlockChan <-chan block.Block
	candidateChan     <-chan message.Candidate
	getCandidateChan  <-chan rpcbus.Request
	bestScoreChan     <-chan message.Message
}

// NewBroker returns an initialized Broker struct. It will still need
// to be started by calling `Listen`.
func NewBroker(broker eventbus.Broker, rpcBus *rpcbus.RPCBus) *Broker {
	acceptedBlockChan, _ := consensus.InitAcceptedBlockUpdate(broker)
	getCandidateChan := make(chan rpcbus.Request, 1)
	_ = rpcBus.Register(topics.GetCandidate, getCandidateChan)
	bestScoreChan := make(chan message.Message, 1)
	broker.Subscribe(topics.BestScore, eventbus.NewChanListener(bestScoreChan))

	b := &Broker{
		publisher:         broker,
		store:             newStore(),
		queue:             make(map[string]message.Candidate),
		validHashes:       make(map[string]struct{}),
		republished:       make(map[string]struct{}),
		acceptedBlockChan: acceptedBlockChan,
		candidateChan:     initCandidateCollector(broker),
		getCandidateChan:  getCandidateChan,
		bestScoreChan:     bestScoreChan,
	}

	broker.Subscribe(topics.ValidCandidateHash, eventbus.NewCallbackListener(b.AddValidHash))
	b.republisher = republisher.New(broker, topics.Candidate, Validate, b.Validate)
	return b
}

// Listen for incoming `Candidate` messages, and internal requests.
// Should be run in a goroutine.
func (b *Broker) Listen() {
	for {
		select {
		case cm := <-b.candidateChan:
			b.lock.Lock()
			b.queue[string(cm.Block.Header.Hash)] = cm
			b.lock.Unlock()
		case r := <-b.getCandidateChan:
			// candidate requests from the RPCBus
			b.provideCandidate(r)
		case blk := <-b.acceptedBlockChan:
			// accepted blocks come from the consensus
			b.lock.Lock()
			b.clearEligibleHashes()
			b.clearRepublishedHashes()
			b.lock.Unlock()
			b.Clear(blk.Header.Height)
		case <-b.bestScoreChan:
			b.filterWinningCandidates()
		}
	}
}

// Filter through the queue with the valid hashes, storing the Candidate
// messages which for which a hash is known, and deleting
// everything else.
func (b *Broker) filterWinningCandidates() {
	b.lock.Lock()
	for k, cm := range b.queue {
		if _, ok := b.validHashes[k]; ok {
			b.storeCandidateMessage(cm)
		}

		delete(b.queue, k)
	}
	b.lock.Unlock()
}

// TODO: interface - rpcBus encoding will be removed
func (b *Broker) provideCandidate(r rpcbus.Request) {
	b.lock.RLock()
	params := r.Params.(bytes.Buffer)
	cm, ok := b.queue[params.String()]
	b.lock.RUnlock()
	if ok {
		r.RespChan <- rpcbus.Response{Resp: cm, Err: nil}
		return
	}

	cm = b.store.fetchCandidateMessage(params.Bytes())
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
	b.publisher.Publish(topics.Gossip, msg)

	timer := time.NewTimer(2 * time.Second)
	for {
		select {
		case <-timer.C:
			return message.Candidate{}, errors.New("request timeout")

		// We take control of `candidateChan`, to monitor incoming
		// candidates. There should be no race condition in reading from
		// the channel, as the only way this function can be called would
		// be through `Listen`. Any incoming candidates which don't match
		// our request will be passed down to the queue.
		case cm := <-b.candidateChan:
			if bytes.Equal(cm.Block.Header.Hash, hash) {
				return cm, nil
			}

			b.lock.Lock()
			b.queue[string(cm.Block.Header.Hash)] = cm
			b.lock.Unlock()
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

// AddValidHash to the local cache for a score. Broker.validHashes uses a map
// to implement a HashSet
func (b *Broker) AddValidHash(m message.Message) error {
	s := m.Payload().(message.Score)
	b.lock.Lock()
	b.validHashes[string(s.VoteHash)] = struct{}{}
	b.lock.Unlock()
	return nil
}

func (b *Broker) clearEligibleHashes() {
	for k := range b.validHashes {
		delete(b.validHashes, k)
	}
}

func (b *Broker) clearRepublishedHashes() {
	for k := range b.republished {
		delete(b.republished, k)
	}
}

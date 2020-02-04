package candidate

import (
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/republisher"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/block"
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
	queue       map[string]Candidate
	validHashes map[string]struct{}

	acceptedBlockChan <-chan block.Block
	candidateChan     <-chan Candidate
	getCandidateChan  <-chan rpcbus.Request
	bestScoreChan     <-chan bytes.Buffer
}

// NewBroker returns an initialized Broker struct. It will still need
// to be started by calling `Listen`.
func NewBroker(broker eventbus.Broker, rpcBus *rpcbus.RPCBus) *Broker {
	acceptedBlockChan, _ := consensus.InitAcceptedBlockUpdate(broker)
	getCandidateChan := make(chan rpcbus.Request, 1)
	rpcBus.Register(rpcbus.GetCandidate, getCandidateChan)
	bestScoreChan := make(chan bytes.Buffer, 1)
	broker.Subscribe(topics.BestScore, eventbus.NewChanListener(bestScoreChan))

	b := &Broker{
		publisher:         broker,
		store:             newStore(),
		queue:             make(map[string]Candidate),
		validHashes:       make(map[string]struct{}),
		acceptedBlockChan: acceptedBlockChan,
		candidateChan:     initCandidateCollector(broker),
		getCandidateChan:  getCandidateChan,
		bestScoreChan:     bestScoreChan,
	}

	broker.Subscribe(topics.ValidCandidateHash, eventbus.NewCallbackListener(b.addValidHash))
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
			b.provideCandidate(r)
		case blk := <-b.acceptedBlockChan:
			b.clearEligibleHashes()
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
		if _, ok := b.validHashes[string(k)]; ok {
			b.storeCandidateMessage(cm)
		}

		delete(b.queue, k)
	}
	b.lock.Unlock()
}

func (b *Broker) provideCandidate(r rpcbus.Request) {
	b.lock.RLock()
	cm, ok := b.queue[string(r.Params.Bytes())]
	b.lock.RUnlock()
	if ok {
		buf := new(bytes.Buffer)
		err := Encode(buf, &cm)
		r.RespChan <- rpcbus.Response{*buf, err}
		return
	}

	cm = b.store.fetchCandidateMessage(r.Params.Bytes())
	if cm.Block == nil {
		// If we don't have the candidate message, we should ask the network for it.
		var err error
		cm, err = b.requestCandidate(r.Params.Bytes())
		if err != nil {
			r.RespChan <- rpcbus.Response{bytes.Buffer{}, err}
			return
		}
	}

	buf := new(bytes.Buffer)
	err := Encode(buf, &cm)
	r.RespChan <- rpcbus.Response{*buf, err}
}

func (b *Broker) requestCandidate(hash []byte) (Candidate, error) {
	// Send a request for this specific candidate
	buf := bytes.NewBuffer(hash)
	if err := topics.Prepend(buf, topics.GetCandidate); err != nil {
		return Candidate{}, err
	}

	b.publisher.Publish(topics.Gossip, buf)

	timer := time.NewTimer(2 * time.Second)
	for {
		select {
		case <-timer.C:
			return Candidate{}, errors.New("request timeout")

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

func (b *Broker) Validate(buf bytes.Buffer) error {
	cm := &Candidate{block.NewBlock(), block.EmptyCertificate()}
	if err := Decode(&buf, cm); err != nil {
		return republisher.EncodingError
	}

	b.lock.RLock()
	defer b.lock.RUnlock()
	if _, ok := b.queue[string(cm.Block.Header.Hash)]; ok {
		return republisher.DuplicatePayloadError
	}

	return nil
}

func (b *Broker) addValidHash(buf bytes.Buffer) error {
	b.lock.Lock()
	b.validHashes[string(buf.Bytes())] = struct{}{}
	b.lock.Unlock()
	return nil
}

func (b *Broker) clearEligibleHashes() {
	b.lock.Lock()
	for k := range b.validHashes {
		delete(b.validHashes, k)
	}
	b.lock.Unlock()
}

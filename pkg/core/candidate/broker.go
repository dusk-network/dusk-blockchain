package candidate

import (
	"bytes"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
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
	// List of block hashes for which a valid Score message was seen.
	validHashes map[string]struct{}

	acceptedBlockChan <-chan block.Block
	candidateChan     <-chan message.Candidate
	getCandidateChan  <-chan rpcbus.Request
}

// NewBroker returns an initialized Broker struct. It will still need
// to be started by calling `Listen`.
func NewBroker(broker eventbus.Broker, rpcBus *rpcbus.RPCBus) *Broker {
	acceptedBlockChan, _ := consensus.InitAcceptedBlockUpdate(broker)
	getCandidateChan := make(chan rpcbus.Request, 1)
	rpcBus.Register(rpcbus.GetCandidate, getCandidateChan)

	b := &Broker{
		publisher:         broker,
		store:             newStore(),
		validHashes:       make(map[string]struct{}),
		acceptedBlockChan: acceptedBlockChan,
		candidateChan:     initCandidateCollector(broker),
		getCandidateChan:  getCandidateChan,
	}

	broker.Subscribe(topics.ValidCandidateHash, eventbus.NewCallbackListener(b.AddValidHash))
	// TODO: interface - uncomment and change into interface/message
	//b.republisher = republisher.New(broker, topics.Candidate, Validate)
	return b
}

// Listen for incoming `Candidate` messages, and internal requests.
// Should be run in a goroutine.
func (b *Broker) Listen() {
	for {
		select {
		case cm := <-b.candidateChan:
			if _, ok := b.validHashes[string(cm.Block.Header.Hash)]; ok {
				b.storeCandidateMessage(cm)
			}
		case r := <-b.getCandidateChan:
			b.provideCandidate(r)
		case blk := <-b.acceptedBlockChan:
			b.clearEligibleBlocks()
			b.Clear(blk.Header.Height)
		}
	}
}

func (b *Broker) AddValidHash(m message.Message) error {
	score := m.Payload().(message.Score)
	b.validHashes[string(score.VoteHash)] = struct{}{}
	return nil
}

// TODO: interface - get rid of encoding if not needed
func (b *Broker) provideCandidate(r rpcbus.Request) {
	cm := b.store.fetchCandidateMessage(r.Params.Bytes())
	if cm == nil {
		// If we don't have the candidate message, we should ask the network for it.
		var err error
		cm, err = b.requestCandidate(r.Params.Bytes())
		if err != nil {
			r.RespChan <- rpcbus.Response{bytes.Buffer{}, err}
			return
		}
	}

	buf := new(bytes.Buffer)
	err := message.MarshalCandidate(buf, *cm)
	r.RespChan <- rpcbus.Response{*buf, err}
}

// requestCandidate from peers around this node. The candidate can only be
// requested for 2 rounds (which provides some protection from keeping to
// request bulky stuff)
func (b *Broker) requestCandidate(hash []byte) (*message.Candidate, error) {
	// Send a request for this specific candidate
	msg := message.New(topics.GetCandidate, bytes.NewBuffer(hash))
	b.publisher.Publish(topics.Gossip, msg)

	timer := time.NewTimer(2 * time.Second)
	for {
		select {
		case <-timer.C:
			return nil, errors.New("request timeout")

		// We take control of `candidateChan`, to monitor incoming
		// candidates. There should be no race condition in reading from
		// the channel, as the only way this function can be called would
		// be through `Listen`. Any incoming candidates which don't match
		// our request will be passed down to the store.
		case cm := <-b.candidateChan:
			// We don't check if this is an eligible candidate block,
			// as we most likely did not get the Score message for it.
			// However, as we are only interested in one specific block,
			// we should not be in danger of memory overflow.
			b.storeCandidateMessage(cm)
			if bytes.Equal(cm.Block.Header.Hash, hash) {
				return &cm, nil
			}
		}
	}
}

func (b *Broker) clearEligibleBlocks() {
	for h := range b.validHashes {
		delete(b.validHashes, h)
	}
}

package candidate

import (
	"bytes"
	"encoding/hex"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
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
	publisher eventbus.Publisher

	*store

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
		acceptedBlockChan: acceptedBlockChan,
		candidateChan:     initCandidateCollector(broker),
		getCandidateChan:  getCandidateChan,
	}
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
			// Remove all candidates with lower or equal round
			b.Clear(blk.Header.Height)
		}
	}
}

// TODO: interface - rpcBus encoding will be removed
func (b *Broker) provideCandidate(r rpcbus.Request) {

	// Read params from request
	params := r.Params.(bytes.Buffer)

	// Mandatory param
	hash := make([]byte, 32)
	if err := encoding.Read256(&params, hash); err != nil {
		r.RespChan <- rpcbus.Response{Resp: nil, Err: err}
		return
	}

	// Enforce fetching from peers if local cache does not have this Candidate
	// Optional param
	var fetchFromPeers bool
	_ = encoding.ReadBool(&params, &fetchFromPeers)

	cm, err := b.store.fetchCandidateMessage(hash)
	if fetchFromPeers && err != nil {
		// If we don't have the candidate message, we should ask the network for it.
		cm, err = b.requestCandidate(hash)
	}

	r.RespChan <- rpcbus.Response{Resp: cm, Err: err}
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

	lg.WithField("hash", hex.EncodeToString(hash)).Trace("Request Candidate from peers")
	// Send a request for this specific candidate
	buf := new(bytes.Buffer)
	_ = encoding.Write256(buf, hash)
	// disable fetching from peers, if not found
	_ = encoding.WriteBool(buf, false)

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
			lg.WithField("hash", hex.EncodeToString(hash)).Trace("Request Candidate from peers failed")
			return message.Candidate{}, ErrGetCandidateTimeout

		// We take control of `candidateChan`, to monitor incoming
		// candidates. There should be no race condition in reading from
		// the channel, as the only way this function can be called would
		// be through `Listen`. Any incoming candidates which don't match
		// our request will be passed down to the queue.
		case cm := <-b.candidateChan:

			b.storeCandidateMessage(cm)

			if bytes.Equal(cm.Block.Header.Hash, hash) {
				lg.WithField("hash", hex.EncodeToString(hash)).Trace("Request Candidate from peers succeeded")
				return cm, nil
			}

		}
	}
}

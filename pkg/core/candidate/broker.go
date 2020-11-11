package candidate

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"
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
	db database.DB
}

// NewBroker returns an initialized Broker struct. It will still need
// to be started by calling `Listen`.
func NewBroker(db database.DB) *Broker {
	return &Broker{
		db: db,
	}
}

// StoreCandidateMessage validates an incoming Candidate message, and then
// stores it in the database if the candidate is valid.
// Satisfies the peer.ProcessorFunc interface.
func (b *Broker) StoreCandidateMessage(msg message.Message) ([]*bytes.Buffer, error) {
	cm := msg.Payload().(message.Candidate)
	if err := ValidateCandidate(cm); err != nil {
		diagnostics.LogError("error in validating the candidate", err)
		return nil, err
	}

	log.Trace("storing candidate message")
	return nil, b.db.Update(func(t database.Transaction) error {
		return t.StoreCandidateMessage(cm)
	})
}

/*
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

	brokergetcandidateTimeout := config.Get().Timeout.TimeoutBrokerGetCandidate
	timer := time.NewTimer(time.Duration(brokergetcandidateTimeout) * time.Second)

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
*/

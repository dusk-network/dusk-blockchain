package peer

import (
	"bytes"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

// The messageRouter is connected to all of the processing units that are tied to the peer.
// It sends an incoming message in the right direction, according to its topic.
type messageRouter struct {
	publisher eventbus.Publisher
	dupeMap   *dupemap.DupeMap

	// 1-to-1 components
	blockHashBroker   *responding.BlockHashBroker
	dataRequestor     *responding.DataRequestor
	dataBroker        *responding.DataBroker
	roundResultBroker *responding.RoundResultBroker
	candidateBroker   *responding.CandidateBroker
	synchronizer      *chainsync.ChainSynchronizer
	ponger            processing.Ponger

	peerInfo string
}

func (m *messageRouter) Collect(packet []byte) error {
	b := bytes.NewBuffer(packet)
	msg, err := message.Unmarshal(b)
	if err != nil {
		return err
	}
	m.route(*b, msg)
	return nil
}

func (m *messageRouter) CanRoute(topic topics.Topic) bool {
	switch topic {
	case topics.Tx,
		topics.Candidate,
		topics.Score,
		topics.Reduction,
		topics.Agreement,
		topics.RoundResults:
		return true
	}

	return false
}

// route accepts a message
// TODO: interface - the re-marshalling and extraction of the Category is a distortion that will
// go away as soon as using messages with struct Payload, instead of Buffer,
// for internal communications
func (m *messageRouter) route(b bytes.Buffer, msg message.Message) {
	var err error
	category := msg.Category()
	switch category {
	case topics.GetBlocks:
		topics.Extract(&b)
		err = m.blockHashBroker.AdvertiseMissingBlocks(&b)
	case topics.GetData:
		topics.Extract(&b)
		err = m.dataBroker.SendItems(&b)
	case topics.MemPool:
		topics.Extract(&b)
		err = m.dataBroker.SendTxsItems()
	case topics.Inv:
		topics.Extract(&b)
		err = m.dataRequestor.RequestMissingItems(&b)
	case topics.Block:
		topics.Extract(&b)
		err = m.synchronizer.Synchronize(&b, m.peerInfo)
	case topics.Ping:
		topics.Extract(&b)
		m.ponger.Pong()
	case topics.Pong:
		// Just here to avoid the error message, as pong is unroutable but
		// otherwise carries no relevant information beyond the receiving
		// of this message
	case topics.GetRoundResults:
		topics.Extract(&b)
		err = m.roundResultBroker.ProvideRoundResult(&b)
	case topics.GetCandidate:
		// We only accept a certain request once, to avoid infinitely
		topics.Extract(&b)
		// requesting the same block
		// TODO: interface - buffer should be immutable. Change the dupemap to
		// deal with values rather than reference
		topics.Extract(&b)
		if m.dupeMap.CanFwd(&b) {
			err = m.candidateBroker.ProvideCandidate(&b)
		}
	default:
		topics.Extract(&b)
		if m.CanRoute(category) {
			if m.dupeMap.CanFwd(&b) {
				m.publisher.Publish(category, msg)
			}
		} else {
			err = fmt.Errorf("%s topic not routable", category.String())
		}
	}

	if err != nil {
		log.WithFields(log.Fields{
			"process": "peer",
			"error":   err,
		}).Errorf("problem handling message %s", category.String())
	}
}

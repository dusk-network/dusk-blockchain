package peer

import (
	"bytes"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
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
	synchronizer      *chainsync.ChainSynchronizer
	ponger            processing.Ponger

	peerInfo string
}

func (m *messageRouter) Collect(b *bytes.Buffer) error {
	topic, err := topics.Extract(b)
	if err != nil {
		return err
	}
	m.route(topic, b)
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

func (m *messageRouter) route(topic topics.Topic, b *bytes.Buffer) {
	var err error
	switch topic {
	case topics.GetBlocks:
		err = m.blockHashBroker.AdvertiseMissingBlocks(b)
	case topics.GetData:
		err = m.dataBroker.SendItems(b)
	case topics.MemPool:
		err = m.dataBroker.SendTxsItems()
	case topics.Inv:
		// We only accept an advertisement once
		if m.dupeMap.CanFwd(b) {
			err = m.dataRequestor.RequestMissingItems(b)
		}
	case topics.Block:
		err = m.synchronizer.Synchronize(b, m.peerInfo)
	case topics.Ping:
		m.ponger.Pong()
	case topics.Pong:
		// Just here to avoid the error message, as pong is unroutable but
		// otherwise carries no relevant information beyond the receiving
		// of this message
	default:
		if m.CanRoute(topic) {
			if m.dupeMap.CanFwd(b) {
				m.publisher.Publish(topic, b)
			}
		} else {
			err = fmt.Errorf("%s topic not routable", topic.String())
		}
	}

	if err != nil {
		log.WithFields(log.Fields{
			"process": "peer",
			"error":   err,
		}).Errorf("problem handling message %s", topic.String())
	}
}

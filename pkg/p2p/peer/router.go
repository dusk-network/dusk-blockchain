package peer

import (
	"bytes"
	"fmt"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/dupemap"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing/chainsync"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// The messageRouter is connected to all of the processing units that are tied to the peer.
// It sends an incoming message in the right direction, according to it's topic.
type messageRouter struct {
	publisher wire.EventPublisher
	dupeMap   *dupemap.DupeMap

	// 1-to-1 components
	blockHashBroker *processing.BlockHashBroker
	dataRequestor   *processing.DataRequestor
	dataBroker      *processing.DataBroker
	synchronizer    *chainsync.ChainSynchronizer

	peerInfo string
}

func (m *messageRouter) Collect(b *bytes.Buffer) error {
	topic, err := extractTopic(b)
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
		topics.Agreement:
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
	default:
		if m.CanRoute(topic) {
			if m.dupeMap.CanFwd(b) {
				m.publisher.Publish(string(topic), b)
			}
		} else {
			err = fmt.Errorf("%s topic not routable", string(topic))
		}
	}

	if err != nil {
		log.WithFields(log.Fields{
			"process": "peer",
			"error":   err,
		}).Errorf("problem handling message %s", string(topic))
	}
}

package peer

import (
	"bytes"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/dupemap"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing/chainsync"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type messageRouter struct {
	publisher wire.EventPublisher
	dupeMap   *dupemap.DupeMap

	// 1-to-1 components
	blockBroker  *processing.BlockBroker
	invBroker    *processing.InvBroker
	dataBroker   *processing.DataBroker
	synchronizer *chainsync.ChainSynchronizer
}

func (m *messageRouter) Collect(b *bytes.Buffer) error {
	if m.dupeMap.CanFwd(b) {
		topic := extractTopic(b)
		m.route(topic, b)
	}
	return nil
}

func (m *messageRouter) route(topic topics.Topic, b *bytes.Buffer) {
	var err error
	switch topic {
	case topics.GetBlocks:
		err = m.blockBroker.AdvertiseMissingBlocks(b)
	case topics.GetData:
		err = m.dataBroker.SendItems(b)
	case topics.Inv:
		err = m.invBroker.AskForMissingItems(b)
	case topics.Block:
		err = m.synchronizer.Synchronize(b)
	default:
		m.publisher.Publish(string(topic), b)
	}

	if err != nil {
		log.WithFields(log.Fields{
			"process": "peer",
			"error":   err,
		}).Errorf("problem handling message %s", string(topic))
	}
}

package peer

import (
	"bytes"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/dupemap"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type messageFilter struct {
	publisher wire.EventPublisher
	dupeMap   *dupemap.DupeMap

	// 1-to-1 components
	blockBroker *blockBroker
	invBroker   *invBroker
	dataBroker  *dataBroker
	blockChan   chan<- *bytes.Buffer
}

func (m *messageFilter) Collect(b *bytes.Buffer) error {
	if m.dupeMap.CanFwd(b) {
		topic := extractTopic(b)
		m.filter(topic, b)
	}
	return nil
}

func (m *messageFilter) filter(topic topics.Topic, b *bytes.Buffer) {

	var err error
	switch topic {
	case topics.GetBlocks:
		err = m.blockBroker.sendBlocks(b)
	case topics.Block:
		m.blockChan <- b
	case topics.GetData:
		err = m.dataBroker.handleMsg(b)
	case topics.Inv:
		err = m.invBroker.handleMsg(b)
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

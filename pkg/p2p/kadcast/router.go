package kadcast

import (
	"bytes"
	"fmt"
	"github.com/dusk-network/dusk-blockchain/pkg/api"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// The messageRouter is connected to all of the processing units that are tied to the peer.
// It sends an incoming message in the right direction, according to its topic.
type messageRouter struct {
	publisher eventbus.Publisher
	dupeMap   *dupemap.DupeMap
}

func (m *messageRouter) Collect(packet []byte, height byte) error {
	b := bytes.NewBuffer(packet)
	msg, err := message.Unmarshal(b)
	if err != nil {
		return err
	}
	return m.route(*b, msg, height)
}

// CanRoute allows only a few wire message that are of broadcast-type.
// For now, request-response wire messages are not introduced in kadcast network
func (m *messageRouter) CanRoute(topic topics.Topic) bool {
	switch topic {
	case topics.Tx, // TODO: Pending
		topics.Block,
		topics.Candidate, // TODO: Pending
		topics.Reduction, // TODO: Pending
		topics.Agreement: // TODO: Pending
		return true
	}

	return false
}

func (m *messageRouter) route(b bytes.Buffer, msg message.Message, height byte) error {

	category := msg.Category()

	log.WithField("topic", category.String()).
		Traceln("peer message received")

	// Filter non-routable messages in kadcast context
	if !m.CanRoute(category) {
		return fmt.Errorf("%s topic not routable", category.String())
	}

	// Filter duplicated messages
	if !m.dupeMap.CanFwd(bytes.NewBuffer(msg.Id())) {
		return fmt.Errorf("duplicated message received with topic %s", category.String())
	}

	// Publish wire message to the eventbus so that the subscribed
	// components could handle it
	switch category {
	case topics.Tx: // TODO:  Not supported yet
	case topics.Candidate: // TODO:  Not supported yet
	case topics.Reduction: // TODO:  Not supported yet
	case topics.Agreement: // TODO:  Not supported yet
	case topics.Block:
		blk := block.NewBlock()
		if err := message.UnmarshalBlock(&b, blk); err != nil {
			return err
		}

		header := []byte{height}
		msg := message.NewWithHeader(topics.Block, *blk, header)
		m.publisher.Publish(category, msg)

		if cfg.Get().API.Prometheus {
			// track processed msg with Prometheus
			api.MsgProcessed.Inc()
		}
	}

	return nil
}

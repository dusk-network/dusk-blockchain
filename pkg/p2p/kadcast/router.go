// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"bytes"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// The messageRouter is connected to all of the processing units that are tied to the peer.
// It sends an incoming message in the right direction, according to its topic.
type messageRouter struct {
	publisher eventbus.Publisher
	dupeMap   *dupemap.DupeMap
}

func (m *messageRouter) Collect(packet []byte, height byte) error {
	// Register message in the global message registry for stats collecting
	diagnostics.RegisterWireMsg(topics.Kadcast.String(), packet)

	b := bytes.NewBuffer(packet)

	msg, err := message.Unmarshal(b)
	if err != nil {
		return err
	}

	// Always append kadcast height
	msg = message.NewWithHeader(msg.Category(), msg.Payload(), []byte{height})
	return m.route(msg)
}

// CanRoute allows only a few wire message that are of broadcast-type.
// For now, request-response wire messages are not introduced in kadcast network.
func (m *messageRouter) CanRoute(topic topics.Topic) bool {
	switch topic {
	case topics.Tx,
		topics.Block,
		topics.Candidate, // TODO: Pending
		topics.Reduction, // TODO: Pending
		topics.Agreement: // TODO: Pending
		return true
	}

	return false
}

func (m *messageRouter) route(msg message.Message) error {
	category := msg.Category()

	log.WithField("topic", category.String()).
		Traceln("peer message received")

	// Filter non-routable messages in kadcast context
	if !m.CanRoute(category) {
		return fmt.Errorf("%s topic not routable", category.String())
	}

	// Filter duplicated messages
	if !m.dupeMap.CanFwd(bytes.NewBuffer(msg.Id())) {
		// return fmt.Errorf("duplicated message received with topic %s", category.String())
	}

	// Publish wire message to the eventbus so that the subscribed
	// components could handle it
	switch category {
	case topics.Tx:
		m.publisher.Publish(category, msg)
	case topics.Candidate:
		// We don't use the dupe map
		// See also topics.Candidate handling in gossip
		m.publisher.Publish(category, msg)
	case topics.Reduction: // TODO:  Not supported yet
	case topics.Agreement: // TODO:  Not supported yet
	case topics.Block:
		m.publisher.Publish(category, msg)
	}

	return nil
}

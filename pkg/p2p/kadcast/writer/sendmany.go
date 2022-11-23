// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package writer

import (
	"context"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// SendToMany collects topics.KadcastSendToMany event to distribute a single
// message to multiple nodes via rusk.NetworkClient Send call.
type SendToMany struct {
	Base
}

// NewSendToMany ...
func NewSendToMany(ctx context.Context, s eventbus.Subscriber, g *protocol.Gossip, rusk rusk.NetworkClient) ring.Writer {
	w := &SendToMany{
		Base: Base{
			subscriber: s,
			gossip:     g,
			client:     rusk,
			ctx:        ctx,
			topic:      topics.KadcastSendToMany,
		},
	}

	w.Subscribe()
	return w
}

// Subscribe subscribes to eventbus Kadcast messages.
func (w *SendToMany) Subscribe() {
	// KadcastPoint subs
	w.subscriptionID = w.subscriber.Subscribe(w.topic, eventbus.NewStreamListener(w))
}

// Write ...
func (w *SendToMany) Write(data []byte, metadata *message.Metadata, priority byte) (int, error) {
	if err := w.sendToMany(data, metadata, priority); err != nil {
		log.WithError(err).Warn("write failed")
	}

	return 0, nil
}

// sendToMany sends a message to N random endpoints returned by AliveNodes.
func (w *SendToMany) sendToMany(data []byte, metadata *message.Metadata, _ byte) error {
	if metadata == nil {
		return errors.New("empty message metadata")
	}

	// get N active nodes
	req := &rusk.AliveNodesRequest{MaxNodes: metadata.NumNodes}

	resp, err := w.client.AliveNodes(w.ctx, req)
	if err != nil {
		log.WithError(err).Warn("get alive nodes failed")
		return err
	}

	for _, addr := range resp.Address {
		_ = w.Send(data, addr)
	}

	return nil
}

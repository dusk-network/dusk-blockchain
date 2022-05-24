// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package writer

import (
	"context"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// SendToOne collects topics.KadcastSendToOne event to distribute a single
// message to a specified node via rusk.NetworkClient Send call.
type SendToOne struct {
	Base
}

// NewSendToOne ...
func NewSendToOne(ctx context.Context, s eventbus.Subscriber, g *protocol.Gossip, rusk rusk.NetworkClient) ring.Writer {
	w := &SendToOne{
		Base: Base{
			subscriber: s,
			gossip:     g,
			client:     rusk,
			ctx:        ctx,
			topic:      topics.KadcastSendToOne,
		},
	}

	w.Subscribe()
	return w
}

// Subscribe subscribes to eventbus Kadcast messages.
func (w *SendToOne) Subscribe() {
	w.subscriptionID = w.subscriber.Subscribe(w.topic, eventbus.NewStreamListener(w))
}

// Write implements. ring.Writer.
func (w *SendToOne) Write(data, header []byte, priority byte) (int, error) {
	if err := w.sendToOne(data, header, priority); err != nil {
		log.WithError(err).Warn("write failed")
	}

	return 0, nil
}

func (w *SendToOne) sendToOne(data, header []byte, _ byte) error {
	if len(header) == 0 {
		return errors.New("empty message header")
	}

	return w.Send(data, string(header))
}

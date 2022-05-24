// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package writer

import (
	"bytes"
	"context"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// Broadcast is a proxy between EventBus and Kadcast service. It subscribes for
// both topics.Kadcast and topics.KadcastPoint, compiles a valid wire frame and
// propagates the message to Kadcast service.
type Broadcast struct {
	Base
}

// NewBroadcast ...
func NewBroadcast(ctx context.Context, s eventbus.Subscriber, g *protocol.Gossip, rusk rusk.NetworkClient) ring.Writer {
	b := &Broadcast{
		Base: Base{
			subscriber: s,
			gossip:     g,
			client:     rusk,
			ctx:        ctx,
			topic:      topics.Kadcast,
		},
	}

	b.Subscribe()
	return b
}

// Subscribe subscribes to eventbus Kadcast messages.
func (w *Broadcast) Subscribe() {
	mapper := func(t topics.Topic) byte {
		const (
			High = byte(1)
			Low  = byte(0)
		)

		switch t {
		case topics.NewBlock, topics.Candidate, topics.GetCandidate,
			topics.Reduction, topics.AggrAgreement, topics.Agreement:
			return High
		default:
			return Low
		}
	}

	// Kadcast subs
	l := eventbus.NewStreamListenerWithParams(w, MaxWriterQueueSize, mapper)
	w.subscriptionID = w.subscriber.Subscribe(w.topic, l)
}

// Write implements. ring.Writer.
func (w *Broadcast) Write(data, header []byte, priority byte) (int, error) {
	if err := w.broadcast(data, header, priority); err != nil {
		// A returned error here is treated as unrecoverable err.
		log.WithError(err).WithField("handler", w.topic.String()).Warn("write failed")
	}

	return 0, nil
}

// broadcast broadcasts message to the entire network.
// The kadcast height is read from message Header.
func (w *Broadcast) broadcast(data, header []byte, _ byte) error {
	// check header
	if len(header) == 0 {
		return errors.New("empty message header")
	}

	// extract kadcast height
	h := uint32(header[0])
	if h == 0 {
		// Apparently, this node is the last peer in a bucket of height 0. We
		// should not repropagate.
		return nil
	}

	// Decrement kadcast height
	h--

	// create the message
	b := bytes.NewBuffer(data)
	if err := w.gossip.Process(b); err != nil {
		return err
	}

	// prepare message
	m := &rusk.BroadcastMessage{
		KadcastHeight: h,
		Message:       b.Bytes(),
	}
	// broadcast message
	if _, err := w.client.Broadcast(w.ctx, m); err != nil {
		log.WithError(err).Warn("failed to broadcast message")
		return err
	}
	return nil
}

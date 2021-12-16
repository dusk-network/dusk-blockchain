// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcli

import (
	"context"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	rusk "github.com/dusk-network/dusk-blockchain/pkg/util/ruskmock/rpc" // mock
	"google.golang.org/grpc"
)

const (
	// MaxWriterQueueSize max number of messages queued for broadcasting.
	// While in Gossip there is a Writer per a peer, in Kadcast there a single Writer.
	// That's why it should be higher than queue size in Gossip.
	MaxWriterQueueSize = 10000
)

// Writer abstracts all of the logic and fields needed to write messages to
// other network nodes.
type Writer struct {
	subscriber eventbus.Subscriber
	gossip     *protocol.Gossip

	kadcastSubscription      uint32
	kadcastPointSubscription uint32

	cli rusk.NetworkClient
}

// NewWriter returns a Writer. It will still need to be initialized by
// subscribing to the gossip topic with a stream handler, and by running the WriteLoop
// in a goroutine..
func NewWriter(subscriber eventbus.Subscriber, gossip *protocol.Gossip, conn *grpc.ClientConn) *Writer {
	return &Writer{
		subscriber: subscriber,
		gossip:     gossip,
		cli:        rusk.NewNetworkClient(conn),
	}
}

// Serve subscribes to eventbus Kadcast messages and injects the writer
func (w *Writer) Serve() error {

	// Kadcast subs
	l1 := eventbus.NewStreamListener(w)
	w.kadcastSubscription = w.subscriber.Subscribe(topics.Kadcast, l1)

	// KadcastPoint subs
	l2 := eventbus.NewStreamListener(w)
	w.kadcastPointSubscription = w.subscriber.Subscribe(topics.KadcastPoint, l2)

	return nil
}

func (w *Writer) Write(data, header []byte, priority byte) (int, error) {
	return 0, nil
}

// WriteToAll broadcasts message to the entire network.
func (w *Writer) WriteToAll(data, header []byte, priority byte) error {
	if len(header) == 0 {
		return errors.New("empty message header")
	}
	height := header[0]
	return w.broadcastPacket(height, data)
}

// WriteToPoint writes a message to a single destination.
// The receiver address is read from message Header.
func (w *Writer) WriteToPoint(data, header []byte, priority byte) error {
	// TODO
	return nil
}

// BroadcastPacket passes a message to the kadkast peer to be broadcasted
func (w *Writer) broadcastPacket(maxHeight byte, payload []byte) error {
	h := uint32(maxHeight)
	m := &rusk.BroadcastMessage{
		KadcastHeight: &h,
		Message:       payload,
	}
	if _, err := w.cli.Broadcast(context.TODO(), m); err != nil {
		log.WithError(err).Warn("failed to broadcast message")
		return err
	}
	return nil
}

// Close unsubscribes from eventbus events.
func (w *Writer) Close() error {
	w.subscriber.Unsubscribe(topics.Kadcast, w.kadcastSubscription)
	w.subscriber.Unsubscribe(topics.KadcastPoint, w.kadcastPointSubscription)
	return nil
}

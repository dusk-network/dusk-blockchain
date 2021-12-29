// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"bytes"
	"context"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// Writer is a proxy between EventBus and Kadcast service. It subscribes for
// both topics.Kadcast and topics.KadcastPoint, compiles a valid wire frame and
// propagates the message to Kadcast service.
type Writer struct {
	subscriber eventbus.Subscriber
	gossip     *protocol.Gossip

	kadcastSubscription      uint32
	kadcastPointSubscription uint32

	client rusk.NetworkClient

	ctx context.Context
}

// NewWriter returns a Writer.
func NewWriter(ctx context.Context, s eventbus.Subscriber, g *protocol.Gossip, rusk rusk.NetworkClient) *Writer {
	return &Writer{
		subscriber: s,
		gossip:     g,
		client:     rusk,
		ctx:        ctx,
	}
}

// Subscribe subscribes to eventbus Kadcast messages.
func (w *Writer) Subscribe() {
	// Kadcast subs
	l1 := eventbus.NewStreamListener(w)
	w.kadcastSubscription = w.subscriber.Subscribe(topics.Kadcast, l1)
	// KadcastPoint subs
	l2 := eventbus.NewStreamListener(w)
	w.kadcastPointSubscription = w.subscriber.Subscribe(topics.KadcastPoint, l2)
}

// Write sends a message through the Kadcast gRPC interface.
// Depending on the value of header field, Send or Broadcast is called.
func (w *Writer) Write(data, header []byte, priority byte) (int, error) {
	// check header
	if len(header) == 0 {
		return 0, errors.New("empty message header")
	}
	// send
	go func() {
		var err error
		// send a p2p message
		if len(header) > 1 {
			err = w.writeToPoint(data, header, priority)
		}
		// broadcast a message
		if len(header) == 1 {
			err = w.writeToAll(data, header, priority)
		}
		// log errors
		if err != nil {
			log.WithError(err).Warn("write failed")
		}
	}()
	return 0, nil
}

// writeToAll broadcasts message to the entire network.
// The kadcast height is read from message Header.
func (w *Writer) writeToAll(data, header []byte, _ byte) error {
	// check header
	if len(header) == 0 {
		return errors.New("empty message header")
	}
	// create the message
	b := bytes.NewBuffer(data)
	if err := w.gossip.Process(b); err != nil {
		return err
	}
	// extract kadcast height
	height := uint32(header[0])
	// prepare message
	m := &rusk.BroadcastMessage{
		KadcastHeight: height,
		Message:       b.Bytes(),
	}
	// broadcast message
	if _, err := w.client.Broadcast(w.ctx, m); err != nil {
		log.WithError(err).Warn("failed to broadcast message")
		return err
	}
	return nil
}

// writeToPoint writes a message to a single destination.
// The receiver address is read from message Header.
func (w *Writer) writeToPoint(data, header []byte, _ byte) error {
	// check header
	if len(header) == 0 {
		return errors.New("empty message header")
	}
	// create the message
	b := bytes.NewBuffer(data)
	if err := w.gossip.Process(b); err != nil {
		return err
	}
	// extract destination address
	addr := string(header)
	// prepare message
	m := &rusk.SendMessage{
		TargetAddress: addr,
		Message:       b.Bytes(),
	}
	// send message
	if _, err := w.client.Send(w.ctx, m); err != nil {
		log.WithError(err).Warn("failed to broadcast message")
		return err
	}
	return nil
}

// Close implements Writer interface.
func (w *Writer) Close() error {
	return nil
}

// Unsubscribe unsubscribes from eventbus events and cancels any remaining writes.
func (w *Writer) Unsubscribe() error {
	w.subscriber.Unsubscribe(topics.Kadcast, w.kadcastSubscription)
	w.subscriber.Unsubscribe(topics.KadcastPoint, w.kadcastPointSubscription)
	return nil
}

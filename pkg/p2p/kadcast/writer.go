// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"

	crypto "github.com/dusk-network/dusk-crypto/hash"
)

const (
	// MaxWriterQueueSize max number of messages queued for broadcasting.
	MaxWriterQueueSize = 1000
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
	l1 := eventbus.NewStreamListenerWithParams(w, MaxWriterQueueSize, mapper)
	w.kadcastSubscription = w.subscriber.Subscribe(topics.Kadcast, l1)

	// KadcastPoint subs
	l2 := eventbus.NewStreamListener(w)
	w.kadcastPointSubscription = w.subscriber.Subscribe(topics.KadcastPoint, l2)
}

// Write sends a message through the Kadcast gRPC interface.
// Depending on the value of header field, Send or Broadcast is called.
func (w *Writer) Write(data, header []byte, priority byte) (int, error) {
	var err error

	switch {
	case len(header) > 1:
		// point-to-point messaging
		err = w.writeToPoint(data, header, priority)
	case len(header) == 1:
		// broadcast messaging
		err = w.writeToAll(data, header, priority)
	default:
		err = errors.New("empty message header")
	}

	// log errors but not return them.
	// A returned error here is treated as unrecoverable err.
	if err != nil {
		log.WithError(err).Warn("write failed")
	}

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
	if height == 0 {
		return nil
	}

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

	// Make the message unique so it is not fitered out by kadcast cache.
	e, _ := crypto.RandEntropy(64)
	reserved := binary.LittleEndian.Uint64(e)

	if err := w.gossip.ProcessWithReserved(b, reserved); err != nil {
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

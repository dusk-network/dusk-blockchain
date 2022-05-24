// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Reader is a proxy between Kadcast grpc service and Message Processor. It
// receives a wire message and ,if it's valid, redirects it to Message Processor.
// In addition, it turns any response message from Processor into
// topics.KadcastPoint EventBus message.
type Reader struct {
	publisher eventbus.Publisher
	processor *peer.MessageProcessor
	gossip    *protocol.Gossip

	client rusk.NetworkClient

	ctx context.Context
}

// NewReader makes a new Kadcast reader.
func NewReader(ctx context.Context, publisher eventbus.Publisher, g *protocol.Gossip, p *peer.MessageProcessor, rusk rusk.NetworkClient) *Reader {
	return &Reader{
		publisher: publisher,
		processor: p,
		gossip:    g,
		client:    rusk,
		ctx:       ctx,
	}
}

// Listen starts accepting and processing stream data.
func (r *Reader) Listen() {
	// create stream handler
	stream, err := r.client.Listen(r.ctx, &rusk.Null{})
	if err != nil {
		log.WithError(err).Fatal("open stream error")
		return
	}

	// listen for messages
	go func() {
		for {
			// receive a message
			msg, err := stream.Recv()
			if err != nil {
				reportStreamErr(err)
				return
			}

			// Message received
			go r.processMessage(msg)
		}
	}()
}

// processMessage propagates the received kadcast message into the event bus.
func (r *Reader) processMessage(msg *rusk.Message) {
	reader := bytes.NewReader(msg.Message)

	// read message (extract length and magic)
	b, err := r.gossip.ReadMessage(reader)
	if err != nil {
		log.WithField("r_addr", msg.Metadata.SrcAddress).
			WithError(err).Warnln("error reading message")
		return
	}
	// extract checksum
	m, cs, err := checksum.Extract(b)
	if err != nil {
		log.WithError(err).Warnln("error extracting message and cs")
		return
	}
	// verify checksum
	if !checksum.Verify(m, cs) {
		log.WithError(errors.New("invalid checksum")).
			Warnln("error verifying message cs")
		return
	}

	h := []byte{byte(msg.Metadata.KadcastHeight)}

	// collect (process) the message
	respBufs, err := r.processor.Collect(msg.Metadata.SrcAddress, m, nil, protocol.FullNode, h)
	if err != nil {
		var topic string
		if len(m) > 0 {
			topic = topics.Topic(m[0]).String()
		}
		// log error
		log.WithField("cs", hex.EncodeToString(cs)).
			WithField("topic", topic).
			WithError(err).Error("failed to process message")
		return
	}
	// any response message is translated into Kadcast Point-to-Point wire message
	// in other words, any bufs item is sent back to the sender node (remotePeer)
	for i := 0; i < len(respBufs); i++ {
		log.WithField("r_addr", msg.Metadata.SrcAddress).Trace("send point-to-point message")
		// send Kadcast point-to-point message with source address as destination
		msg := message.NewWithHeader(topics.KadcastSendToOne, respBufs[i], []byte(msg.Metadata.SrcAddress))
		r.publisher.Publish(topics.KadcastSendToOne, msg)
	}
}

func reportStreamErr(err error) {
	m := "listener_loop terminated"

	s, ok := status.FromError(err)
	if ok {
		switch s.Code() {
		case codes.Canceled:
			// canceling
			log.Info(m)

		default:
			log.
				WithField("code", s.Code()).
				WithError(err).Warn(m)
		}
		return
	}

	log.WithError(err).Error(m)
}

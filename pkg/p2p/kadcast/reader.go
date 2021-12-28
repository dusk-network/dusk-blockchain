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
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// Reader abstracts all of the logic and fields needed to receive messages from
// other network nodes.
type Reader struct {
	publisher eventbus.Publisher
	processor *peer.MessageProcessor
	gossip    *protocol.Gossip

	client rusk.NetworkClient
	stop   context.CancelFunc
}

// NewReader makes a new kadcast reader that handles TCP packets of broadcasting.
func NewReader(publisher eventbus.Publisher, g *protocol.Gossip, p *peer.MessageProcessor, rusk rusk.NetworkClient) *Reader {
	return &Reader{
		publisher: publisher,
		processor: p,
		gossip:    g,
		client:    rusk,
	}
}

// Listen starts accepting and processing stream data.
func (r *Reader) Listen() {
	// create stream handler
	stream, err := r.client.Listen(context.Background(), &rusk.Null{})
	if err != nil {
		log.WithError(err).Fatal("open stream error")
		return
	}

	// create a context
	ctx, cancel := context.WithCancel(context.Background())
	r.stop = cancel

	// listen for messages
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// receive a message
				msg, err := stream.Recv()
				if err == io.EOF {
					log.Warnln("reader reached EOF")
					return
				} else if err != nil {
					log.WithError(err).Fatal("receive error")
				}
				// Message received
				go r.processMessage(msg)
			}
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

	// collect (process) the message
	go func() {
		respBufs, err := r.processor.Collect(msg.Metadata.SrcAddress, m, nil, protocol.FullNode, []byte{byte(msg.Metadata.KadcastHeight)})
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
			msg := message.NewWithHeader(topics.KadcastPoint, respBufs[i], []byte(msg.Metadata.SrcAddress))
			r.publisher.Publish(topics.KadcastPoint, msg)
		}
	}()
}

// Close closes reader TCP listener.
func (r *Reader) Close() error {
	r.stop()
	return nil
}

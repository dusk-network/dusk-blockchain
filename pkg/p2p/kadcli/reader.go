// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcli

import (
	"context"
	"encoding/hex"
	"errors"
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"google.golang.org/grpc"
)

// Reader abstracts all of the logic and fields needed to receive messages from
// other network nodes.
type Reader struct {
	processor *peer.MessageProcessor

	cli  rusk.NetworkClient
	stop chan bool
}

// NewReader makes a new kadcast reader that handles TCP packets of broadcasting.
func NewReader(publisher eventbus.Publisher, processor *peer.MessageProcessor, ruskConn *grpc.ClientConn) *Reader {
	return &Reader{
		processor: processor,
		cli:       rusk.NewNetworkClient(ruskConn),
		stop:      make(chan bool),
	}
}

// Serve starts accepting and processing stream data.
func (r *Reader) Serve() {
	// create stream handler
	stream, err := r.cli.Listen(context.Background(), &rusk.Null{})
	if err != nil {
		log.Fatalf("open stream error %v", err)
		return
	}

	// create stop channel
	r.stop = make(chan bool)

	// listen for messages
	go func() {
		// TODO: deadline/ttl?
		for {
			select {
			case <-r.stop:
				return
			default:
				// receive a message
				msg, err := stream.Recv()
				if err == io.EOF {
					// stream is done
					// TODO: Notify someone? Retry mechanism?
					log.Panic("stream done (eof)")
					return
				} else if err != nil {
					log.Fatalf("recv error %v", err)
				}
				// Message received
				go r.processMessage(msg)
				log.Infof("received msg: %v", msg)
			}
		}
	}()
}

// processMessage propagates the received kadcast message into the event bus.
func (r *Reader) processMessage(message *rusk.Message) {
	// Extract checksum
	msg, cs, err := checksum.Extract(message.Message)
	if err != nil {
		log.WithError(err).Warnln("error extracting message and cs")
		return
	}
	// Verify checksum
	if !checksum.Verify(msg, cs) {
		log.WithError(errors.New("invalid checksum")).
			Warnln("error verifying message cs")
		return
	}
	// Collect (and process) the message
	go func() {
		if _, err = r.processor.Collect(message.Metadata.SrcAddress, message.Message, nil, protocol.FullNode, nil); err != nil {
			var topic string
			if len(msg) > 0 {
				topic = topics.Topic(msg[0]).String()
			}
			// log error
			log.WithField("cs", hex.EncodeToString(cs)).
				WithField("topic", topic).
				WithError(err).Error("failed to process message")
		}
	}()
}

// Close closes reader TCP listener.
func (r *Reader) Close() error {
	r.stop <- true
	close(r.stop)
	return nil
}

// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcli

import (
	"context"
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"google.golang.org/grpc"
)

type Reader struct {
	cli  rusk.NetworkClient
	stop chan bool
}

// NewReader makes a new kadcast reader that handles TCP packets of broadcasting.
func NewReader(publisher eventbus.Publisher, processor *peer.MessageProcessor, ruskConn *grpc.ClientConn) *Reader {
	return &Reader{
		cli:  rusk.NewNetworkClient(ruskConn),
		stop: make(chan bool),
	}
	return nil
}

// Serve starts accepting and processing stream data.
func (r *Reader) Serve() {
	// create stream handler
	stream, err := r.cli.Listen(context.Background(), &rusk.Null{})
	if err != nil {
		log.Fatalf("open stream error %v", err)
	}

	// create stop channel
	r.stop = make(chan bool)

	// listen for messages
	go func() {
		for {
			select {
			case <-r.stop:
				return
			default:
				// receive a message
				msg, err := stream.Recv()
				if err == io.EOF {
					// stream is done
					// TODO: Notify someone?
					log.Info("stream done")
					return
				} else if err != nil {
					log.Fatalf("recv error %v", err)
				}
				// Message received
				log.Infof("received msg: %v", msg)
				go r.processMessage(msg)
			}
		}
	}()
}

// processMessage propagates the received kadcast message into the event bus
func (r *Reader) processMessage(msg *rusk.Message) {
	// TODO
}

// Close closes reader TCP listener.
func (r *Reader) Close() error {
	r.stop <- true
	close(r.stop)
	return nil
}

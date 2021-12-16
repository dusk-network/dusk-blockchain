// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcli

import (
	"net"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type Reader struct {
	listener *net.TCPListener
}

// NewReader makes a new kadcast reader that handles TCP packets of broadcasting.
func NewReader(publisher eventbus.Publisher, gossip *protocol.Gossip, processor *peer.MessageProcessor) *Reader {
	// https://www.freecodecamp.org/news/grpc-server-side-streaming-with-go/
	return nil
}

// Close closes reader TCP listener.
func (r *Reader) Close() error {
	if r.listener != nil {
		return r.listener.Close()
	}

	return nil
}

// Serve starts accepting and processing TCP connection and packets.
func (r *Reader) Serve() {}

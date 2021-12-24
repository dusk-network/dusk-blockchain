// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcli

import (
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logger.WithFields(logger.Fields{"process": "kadcli"})

// Peer is a wrapper for both kadcast grpc sides.
type Peer struct {
	// dusk node components
	eventBus  *eventbus.EventBus
	processor *peer.MessageProcessor
	gossip    *protocol.Gossip

	// processors
	w *Writer
	r *Reader
}

// NewCliPeer makes a kadcli peer instance.
func NewCliPeer(eventBus *eventbus.EventBus, processor *peer.MessageProcessor, gossip *protocol.Gossip) *Peer {
	return &Peer{eventBus: eventBus, processor: processor, gossip: gossip}
}

// Launch starts kadcli service and connects to server.
func (p *Peer) Launch(conn *grpc.ClientConn) {
	// gRPC rusk client
	ruskc := rusk.NewNetworkClient(conn)
	// a writer for Kadcast messages
	p.w = NewWriter(p.eventBus, ruskc)
	go p.w.Serve()
	// a reader for Kadcast messages
	p.r = NewReader(p.eventBus, p.gossip, p.processor, ruskc)
	go p.r.Listen()
}

// Close terminates kadcli service.
func (p *Peer) Close() {
	// close writer
	if p.w != nil {
		_ = p.w.Close()
	}
	// close reader
	if p.r != nil {
		_ = p.r.Close()
	}
}

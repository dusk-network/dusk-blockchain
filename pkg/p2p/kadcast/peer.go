// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logger.WithFields(logger.Fields{"process": "kadcast"})

// Peer is a wrapper for both kadcast grpc sides.
type Peer struct {
	// dusk node components
	eventBus  *eventbus.EventBus
	processor *peer.MessageProcessor
	gossip    *protocol.Gossip

	// processors
	w *Writer
	r *Reader

	ctx    context.Context
	cancel context.CancelFunc
}

// NewKadcastPeer returns a new kadcast (gRPC interface) peer instance.
func NewKadcastPeer(eventBus *eventbus.EventBus, processor *peer.MessageProcessor, gossip *protocol.Gossip) *Peer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Peer{
		eventBus:  eventBus,
		processor: processor,
		gossip:    gossip,
		cancel:    cancel,
		ctx:       ctx,
	}
}

// Launch starts kadcast peer reader and writers, binds them to the event buss,
// and establishes connection to rusk network server.
func (p *Peer) Launch(conn *grpc.ClientConn) {
	// gRPC rusk client
	ruskc := rusk.NewNetworkClient(conn)
	// a writer for Kadcast messages
	p.w = NewWriter(p.ctx, p.eventBus, p.gossip, ruskc)
	p.w.Subscribe()
	// a reader for Kadcast messages
	p.r = NewReader(p.ctx, p.eventBus, p.gossip, p.processor, ruskc)
	go p.r.Listen()
}

// Close terminates kadcast peer instance.
func (p *Peer) Close() {
	if p.ctx != nil {
		p.cancel()
	}

	// close writer
	if p.w != nil {
		_ = p.w.Unsubscribe()
	}
}

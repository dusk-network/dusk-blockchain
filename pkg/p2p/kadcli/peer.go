// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcli

import (
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logger.WithFields(logger.Fields{"process": "kadcli"})

// Peer is a wrapper for both kadcast grpc sides.
type Peer struct {
	// dusk node components
	eventBus  *eventbus.EventBus
	processor *peer.MessageProcessor

	// rusk connection
	conn *grpc.ClientConn

	// processors
	w *Writer
	r *Reader
}

// NewCliPeer makes a kadcli peer instance.
func NewCliPeer(eventBus *eventbus.EventBus, processor *peer.MessageProcessor, ruskConn *grpc.ClientConn) *Peer {
	return &Peer{eventBus: eventBus, processor: processor, conn: ruskConn}
}

// Launch starts kadcli service and connects to server.
func (p *Peer) Launch() {
	// A writer for Kadcast messages
	p.w = NewWriter(p.eventBus, p.conn)
	go p.w.Serve()
	// A reader for Kadcast messages
	p.r = NewReader(p.eventBus, p.processor, p.conn)
	go p.r.Serve()
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

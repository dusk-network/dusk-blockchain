// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"context"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
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

	rConn, wConn *grpc.ClientConn

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
func (p *Peer) Launch() {
	// gRPC rusk client
	cfg := config.Get().Kadcast
	addr := fmt.Sprintf("%s:%d", cfg.GrpcHost, cfg.GrpcPort)
	log.WithField("grpc_addr", addr).Info("launch peer connections")

	// a writer for Kadcast messages
	writerClient, wConn := createNetworkClient(p.ctx, addr)
	p.w = NewWriter(p.ctx, p.eventBus, p.gossip, writerClient)
	p.w.Subscribe()

	// a reader for Kadcast messages
	readerClient, rConn := createNetworkClient(p.ctx, addr)
	p.r = NewReader(p.ctx, p.eventBus, p.gossip, p.processor, readerClient)

	go p.r.Listen()

	p.rConn = rConn
	p.wConn = wConn
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

	if p.rConn != nil {
		_ = p.rConn.Close()
	}

	if p.wConn != nil {
		_ = p.wConn.Close()
	}

	log.Info("peer closed")
}

// createNetworkClient creates a client for the Kadcast network layer.
func createNetworkClient(ctx context.Context, address string) (rusk.NetworkClient, *grpc.ClientConn) {
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Panic(err)
	}

	return rusk.NewNetworkClient(conn), conn
}

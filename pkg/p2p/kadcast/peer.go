// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"context"
	"time"

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
func NewKadcastPeer(pCtx context.Context, eventBus *eventbus.EventBus, processor *peer.MessageProcessor, gossip *protocol.Gossip) *Peer {
	ctx, cancel := context.WithCancel(pCtx)
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

	log.WithField("grpc_addr", cfg.Grpc.Address).
		WithField("grpc_network", cfg.Grpc.Network).
		Info("launch peer connections")

	// a writer for Kadcast messages
	writerClient, wConn := CreateNetworkClient(p.ctx, cfg.Grpc.Network, cfg.Grpc.Address, cfg.Grpc.DialTimeout)
	p.w = NewWriter(p.ctx, p.eventBus, p.gossip, writerClient)
	p.w.Subscribe()

	// a reader for Kadcast messages
	readerClient, rConn := CreateNetworkClient(p.ctx, cfg.Grpc.Network, cfg.Grpc.Address, cfg.Grpc.DialTimeout)
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

// CreateNetworkClient creates a client for the Kadcast network layer.
func CreateNetworkClient(ctx context.Context, network, address string, dialTimeout int) (rusk.NetworkClient, *grpc.ClientConn) {
	var prefix string

	switch network {
	case "tcp":
		prefix = ""
	case "unix":
		prefix = "unix://"
	default:
		panic("unsupported network " + network)
	}

	dialCtx, cancel := context.WithTimeout(ctx, time.Duration(dialTimeout)*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, prefix+address, grpc.WithInsecure(), grpc.WithAuthority("dummy"), grpc.WithBlock())
	if err != nil {
		log.Panic(err)
	}

	return rusk.NewNetworkClient(conn), conn
}

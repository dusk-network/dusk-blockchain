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
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/writer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var log = logger.WithFields(logger.Fields{"process": "kadcast"})

// Peer is a wrapper for both kadcast grpc sides.
type Peer struct {
	// dusk node components
	eventBus  *eventbus.EventBus
	processor *peer.MessageProcessor
	gossip    *protocol.Gossip

	// processors
	writers []ring.Writer
	reader  *Reader

	connections []*grpc.ClientConn

	ctx    context.Context
	cancel context.CancelFunc
}

// NewKadcastPeer returns a new kadcast (gRPC interface) peer instance.
func NewKadcastPeer(pCtx context.Context, eventBus *eventbus.EventBus, processor *peer.MessageProcessor, gossip *protocol.Gossip) *Peer {
	ctx, cancel := context.WithCancel(pCtx)
	return &Peer{
		eventBus:    eventBus,
		processor:   processor,
		gossip:      gossip,
		cancel:      cancel,
		ctx:         ctx,
		connections: make([]*grpc.ClientConn, 0),
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

	// initiate all writers for Kadcast messages.
	p.createWriters()

	// a reader for Kadcast messages
	client, conn := CreateNetworkClient(p.ctx, cfg.Grpc.Network, cfg.Grpc.Address, cfg.Grpc.DialTimeout)
	p.reader = NewReader(p.ctx, p.eventBus, p.gossip, p.processor, client)

	p.connections = append(p.connections, conn)

	go p.reader.Listen()
}

func (p *Peer) createWriters() {
	cfg := config.Get().Kadcast

	// Broadcast
	client, conn := CreateNetworkClient(p.ctx, cfg.Grpc.Network, cfg.Grpc.Address, cfg.Grpc.DialTimeout)
	w := writer.NewBroadcast(p.ctx, p.eventBus, p.gossip, client)
	p.connections = append(p.connections, conn)
	p.writers = append(p.writers, w)

	// Send to One
	client, conn = CreateNetworkClient(p.ctx, cfg.Grpc.Network, cfg.Grpc.Address, cfg.Grpc.DialTimeout)
	w = writer.NewSendToOne(p.ctx, p.eventBus, p.gossip, client)
	p.connections = append(p.connections, conn)
	p.writers = append(p.writers, w)

	// Send to Many
	client, conn = CreateNetworkClient(p.ctx, cfg.Grpc.Network, cfg.Grpc.Address, cfg.Grpc.DialTimeout)
	w = writer.NewSendToMany(p.ctx, p.eventBus, p.gossip, client)
	p.connections = append(p.connections, conn)
	p.writers = append(p.writers, w)
}

// Close terminates kadcast peer instance.
func (p *Peer) Close() {
	if p.ctx != nil {
		p.cancel()
	}

	// close writer
	for _, w := range p.writers {
		_ = w.Close()
	}

	for _, conn := range p.connections {
		if conn != nil {
			_ = conn.Close()
		}
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

	ctxWithVersion := InjectRuskVersion(ctx)

	dialCtx, cancel := context.WithTimeout(ctxWithVersion, time.Duration(dialTimeout)*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, prefix+address, grpc.WithInsecure(), grpc.WithAuthority("dummy"), grpc.WithBlock())
	if err != nil {
		log.Panic(err)
	}

	return rusk.NewNetworkClient(conn), conn
}

// InjectRuskVersion injects the rusk version into the grpc headers.
func InjectRuskVersion(ctx context.Context) context.Context {
	md := metadata.New(map[string]string{"x-rusk-version": config.RuskVersion})
	return metadata.NewOutgoingContext(ctx, md)
}

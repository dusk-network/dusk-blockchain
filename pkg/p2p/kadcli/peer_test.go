// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"google.golang.org/grpc"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/client"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/stretchr/testify/assert"
)

const (
	RUSK_ADDR = "127.0.0.1:50005"
)

// NetworkServer mocks the Rusk network server.
type NetworkServer struct {
	g *protocol.Gossip
}

// Listen for messages coming from the network.
func (n *NetworkServer) Listen(in *rusk.Null, srv rusk.Network_ListenServer) error {
	buf := new(bytes.Buffer)

	// Using a block as an example
	blk := helper.RandomBlock(5525, 10)
	if err := message.MarshalBlock(buf, blk); err != nil {
		return fmt.Errorf("failed to marshal block: %v", err)
	}

	// Add topic to message
	if err := topics.Prepend(buf, topics.Block); err != nil {
		return fmt.Errorf("failed to add topic: %v", err)
	}

	// Make it protocol-ready
	if err := n.g.Process(buf); err != nil {
		return fmt.Errorf("failed to process (gossip): %v", err)
	}

	// Send message
	msg := &rusk.Message{
		Message: buf.Bytes(),
		Metadata: &rusk.MessageMetadata{
			KadcastHeight: uint32(1),
			SrcAddress:    RUSK_ADDR,
		},
	}
	if err := srv.Send(msg); err != nil {
		return fmt.Errorf("failed to send msg: %v", err.Error())
	}
	return nil
}

// Broadcast a message to the network.
func (n *NetworkServer) Broadcast(ctx context.Context, msg *rusk.BroadcastMessage) (*rusk.Null, error) {
	return nil, nil
}

// Send a message to a specific target in the network.
func (n *NetworkServer) Send(ctx context.Context, msg *rusk.SendMessage) (*rusk.Null, error) {
	return nil, nil
}

// NewRuskMock creates a Rusk Network server mock for tests.
func NewRuskMock(gossip *protocol.Gossip, errChan chan error) (*NetworkServer, error) {
	// create listener
	l, err := net.Listen("tcp", RUSK_ADDR)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	// create grpc server
	s := grpc.NewServer()
	srv := &NetworkServer{
		g: gossip,
	}
	rusk.RegisterNetworkServer(s, srv)

	// and start mock service
	go func() {
		if err := s.Serve(l); err != nil {
			errChan <- fmt.Errorf("failed to start grpc server: %v", err)
			return
		}
	}()

	return srv, nil
}

// TestListenStreamReader tests the kadcli.Reader receiving a block
// from the rusk:Network.Listen stream-based endpoint.
func TestListenStreamReader(t *testing.T) {
	assert := assert.New(t)
	rcvChan := make(chan message.Message)
	errChan := make(chan error)

	// Basic infrastructure
	eb := eventbus.New()
	p := peer.NewMessageProcessor(eb)
	g := protocol.NewGossip(protocol.TestNet)

	// Have a callback ready for Block topic
	respFn := func(srcPeerID string, m message.Message) ([]bytes.Buffer, error) {
		if srcPeerID != RUSK_ADDR {
			errChan <- errors.New("callback: wrong source address")
			return nil, nil
		}
		rcvChan <- m
		return nil, nil
	}
	p.Register(topics.Block, respFn)

	// Create a rusk mock server
	_, err := NewRuskMock(g, errChan)
	if err != nil {
		t.Fatalf("rusk mock failed: %v", err)
	}

	// Create a client that connects to rusk mock server
	_, grpConn := client.CreateNetworkClient(context.Background(), RUSK_ADDR)

	// Create our kadcli (gRPC) Reader
	r := NewReader(eb, g, p, grpConn)

	// Subscribe to gRPC stream
	go r.Listen()

	// Process status/output
	select {
	case err := <-errChan:
		t.Fatal(err)
	case m := <-rcvChan:
		b := m.Payload().(block.Block)
		assert.True(m.Category() == topics.Block)
		assert.True(b.Header.Height == 5525)
		assert.True(len(b.Txs) == 12)
	}
}

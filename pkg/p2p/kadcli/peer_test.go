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
	bcastChan chan *rusk.BroadcastMessage
}

// Listen for messages coming from the network.
func (n *NetworkServer) Listen(in *rusk.Null, srv rusk.Network_ListenServer) error {
	go func() {
		for {
			select {
			case msg := <-n.bcastChan:
				m := &rusk.Message{
					Message: msg.Message,
					Metadata: &rusk.MessageMetadata{
						KadcastHeight: msg.KadcastHeight,
						SrcAddress:    RUSK_ADDR,
					},
				}
				if err := srv.Send(m); err != nil {
					fmt.Println("failed to send msg: " + err.Error())
				}
				return
			}
		}
	}()
	return nil
}

// Broadcast a message to the network.
func (n *NetworkServer) Broadcast(ctx context.Context, msg *rusk.BroadcastMessage) (*rusk.Null, error) {
	res := &rusk.Null{}
	fmt.Println("server.Broadcast: ", msg.KadcastHeight)

	// send the message to all listeners
	n.bcastChan <- msg

	return res, nil
}

// Send a message to a specific target in the network.
func (n *NetworkServer) Send(ctx context.Context, msg *rusk.SendMessage) (*rusk.Null, error) {
	fmt.Println("server.Send: ", msg.TargetAddress)
	res := &rusk.Null{}
	return res, nil
}

// NewRuskMock creates a Rusk Network server mock for tests.
func NewRuskMock(gossip *protocol.Gossip, errChan chan error) (*grpc.Server, error) {
	// create listener
	l, err := net.Listen("tcp", RUSK_ADDR)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	// create grpc server
	s := grpc.NewServer()
	srv := &NetworkServer{
		bcastChan: make(chan *rusk.BroadcastMessage),
	}
	rusk.RegisterNetworkServer(s, srv)

	// and start mock service
	go func() {
		if err := s.Serve(l); err != nil {
			errChan <- fmt.Errorf("failed to start grpc server: %v", err)
			return
		}
	}()

	return s, nil
}

// TestListenStreamReader tests the kadcli.Reader receiving a block
// from the rusk:Network.Listen stream-based endpoint.
func TestFull(t *testing.T) {
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
	srv, err := NewRuskMock(g, errChan)
	if err != nil {
		t.Fatalf("rusk mock failed: %v", err)
	}

	// Create a client that connects to rusk mock server
	_, grpcConn := client.CreateNetworkClient(context.Background(), RUSK_ADDR)

	// Create our kadcli (gRPC) Reader and Writer instances
	r := NewReader(eb, g, p, grpcConn)
	w := NewWriter(eb, g, grpcConn)

	// Subscribe to gRPC stream
	go r.Listen()

	// Prepare a message to be sent
	buf, err := createBlockMessage()
	if err != nil {
		t.Errorf("block msg: %v", err)
	}

	// Send a broadcast message
	if err := w.WriteToAll(buf.Bytes(), []byte{127}, 0); err != nil {
		t.Errorf("failed to broadcast: %v", err)
	}

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

	// Clean up connection
	grpcConn.Close()
	srv.Stop()

}

// createBlockMessage returns a properly encoded wire message
// containing a random block
func createBlockMessage() (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	g := protocol.NewGossip(protocol.TestNet)
	// Create the random block and marshal it
	blk := helper.RandomBlock(5525, 10)
	if err := message.MarshalBlock(buf, blk); err != nil {
		return nil, fmt.Errorf("failed to marshal block: %v", err)
	}
	// Add topic to message
	if err := topics.Prepend(buf, topics.Block); err != nil {
		return nil, fmt.Errorf("failed to add topic: %v", err)
	}
	// Make it protocol-ready
	if err := g.Process(buf); err != nil {
		return nil, fmt.Errorf("failed to process (gossip): %v", err)
	}
	return buf, nil
}

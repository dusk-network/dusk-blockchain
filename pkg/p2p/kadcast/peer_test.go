// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

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
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
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

// TestListenStreamReader tests the kadcli.Reader receiving a block
// from the rusk:Network.Listen stream-based endpoint.
func TestListenStreamReader(t *testing.T) {
	assert := assert.New(t)
	rcvChan := make(chan message.Message)
	errChan := make(chan error)

	// basic infrastructure
	eb := eventbus.New()
	p := peer.NewMessageProcessor(eb)
	g := protocol.NewGossip()

	// have a callback ready for Block topic
	respFn := func(srcPeerID string, m message.Message) ([]bytes.Buffer, error) {
		if srcPeerID != RUSK_ADDR {
			errChan <- errors.New("callback: wrong source address")
			return nil, nil
		}
		rcvChan <- m
		return nil, nil
	}
	p.Register(topics.Block, respFn)

	// create a rusk mock server
	srv, err := NewRuskMock(g, errChan)
	if err != nil {
		t.Fatalf("rusk mock failed: %v", err)
	}

	// create a client that connects to rusk mock server
	_, grpcConn := client.CreateNetworkClient(context.Background(), RUSK_ADDR)
	ruskc := rusk.NewNetworkClient(grpcConn)

	// create our kadcli (gRPC) Reader
	r := NewReader(context.Background(), eb, g, p, ruskc)

	// subscribe to gRPC stream
	go r.Listen()

	// process status/output
	select {
	case err := <-errChan:
		t.Fatal(err)
	case m := <-rcvChan:
		b := m.Payload().(block.Block)
		assert.True(m.Category() == topics.Block)
		assert.True(b.Header.Height == 5525)
		assert.True(len(b.Txs) == 11)
	}

	// Disconnect client and stop server
	grpcConn.Close()
	srv.Stop()
}

// TestBroadcastWriter tests the kadcli.Writer by broadcasting
// a block message through a mocked rusk client.
func TestBroadcastWriter(t *testing.T) {
	assert := assert.New(t)
	rcvChan := make(chan *rusk.BroadcastMessage)

	// Basic infrastructure
	eb := eventbus.New()
	g := protocol.NewGossip()

	// create a mock client
	cli := NewMockNetworkClient(rcvChan)

	// create our kadcli Writer
	w := NewWriter(context.Background(), eb, g, cli)
	w.Subscribe()

	// create a mock message
	buf, err := createBlockMessage()
	if err != nil {
		t.Errorf("fail to create msg: %v", err)
	}

	// send a broadcast message
	pubm := message.NewWithHeader(topics.Block, *buf, []byte{127})

	errList := eb.Publish(topics.Kadcast, pubm)
	if len(errList) > 0 {
		t.Fatal("error publishing to evt bus")
	}

	// process status/output
	m := <-rcvChan
	assert.True(m.KadcastHeight == 127)

	// attempt to read the message
	reader := bytes.NewReader(m.Message)

	// read message (extract length and magic)
	b, err := g.ReadMessage(reader)
	if err != nil {
		t.Errorf("error reading message: %v", err)
	}
	// extract checksum
	msg, cs, err := checksum.Extract(b)
	if err != nil {
		t.Errorf("error extracting checksum: %v", err)
	}
	// verify checksum
	if !checksum.Verify(msg, cs) {
		t.Error("invalid checksum")
	}
	// validate message content
	if len(msg) == 0 {
		t.Error("empty packet received")
	}
	// check topic
	rb := bytes.NewBuffer(msg)
	topic := topics.Topic(rb.Bytes()[0])
	assert.True(topic == topics.Block)
	// unmarshal message
	res, err := message.Unmarshal(rb, []byte{})
	if err != nil {
		t.Error("failed to unmarshal")
	}
	// check payload
	if _, ok := res.Payload().(block.Block); !ok {
		t.Error("failed to cast to block")
	}
}

//
// Mock types
//

// NetworkServer mocks the Rusk network server to test the
// reader implementation when connected to the stream.
type MockNetworkServer struct {
	g *protocol.Gossip
}

// Listen for messages coming from the network.
func (n *MockNetworkServer) Listen(in *rusk.Null, srv rusk.Network_ListenServer) error {
	// create a mock message
	buf, err := createBlockMessage()
	if err != nil {
		return fmt.Errorf("fail to create msg: %v", err)
	}
	// make it protocol-ready
	if err := n.g.Process(buf); err != nil {
		return fmt.Errorf("failed to process (gossip): %v", err)
	}
	// prepare the grpc message
	msg := &rusk.Message{
		Message: buf.Bytes(),
		Metadata: &rusk.MessageMetadata{
			KadcastHeight: uint32(1),
			SrcAddress:    RUSK_ADDR,
		},
	}

	// send it back to the client
	if err := srv.Send(msg); err != nil {
		return fmt.Errorf("failed to send msg: %v", err.Error())
	}
	return nil
}

// Broadcast is just here to satisfy the interface. Not used.
func (n *MockNetworkServer) Broadcast(ctx context.Context, msg *rusk.BroadcastMessage) (*rusk.Null, error) {
	res := &rusk.Null{}
	return res, errors.New("not implemented")
}

// Send is just here to satisfy the interface. Not used.
func (n *MockNetworkServer) Send(ctx context.Context, msg *rusk.SendMessage) (*rusk.Null, error) {
	res := &rusk.Null{}
	return res, errors.New("not implemented")
}

// Propagate is just here to satisfy the interface. Not used.
func (n *MockNetworkServer) Propagate(ctx context.Context, msg *rusk.PropagateMessage) (*rusk.Null, error) {
	res := &rusk.Null{}
	return res, errors.New("not implemented")
}

// AliveNodes is just here to satisfy the interface. Not used.
func (n *MockNetworkServer) AliveNodes(ctx context.Context, in *rusk.AliveNodesRequest) (*rusk.AliveNodesResponse, error) {
	res := &rusk.AliveNodesResponse{}
	return res, errors.New("not implemented")
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
	srv := &MockNetworkServer{
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

	return s, nil
}

// MockNetworkClient is used to test the Writer implementation
// by mocking the Broadcast message and redirecting it's output
// to a channel we can receive locally.
type MockNetworkClient struct {
	msgChan chan *rusk.BroadcastMessage
}

// NewMockNetworkClient returns a new instance of a mock network client.
func NewMockNetworkClient(msgChan chan *rusk.BroadcastMessage) *MockNetworkClient {
	return &MockNetworkClient{
		msgChan: msgChan,
	}
}

// Broadcast will check the message is formatted properly.
func (c *MockNetworkClient) Broadcast(ctx context.Context, in *rusk.BroadcastMessage, opts ...grpc.CallOption) (*rusk.Null, error) {
	// send message back
	c.msgChan <- in
	// return
	res := &rusk.Null{}
	return res, nil
}

// Listen is just here to satisfy the interface. Not used.
func (c *MockNetworkClient) Listen(ctx context.Context, in *rusk.Null, opts ...grpc.CallOption) (rusk.Network_ListenClient, error) {
	return nil, errors.New("not implemented")
}

// Send is just here to satisfy the interface. Not used.
func (c *MockNetworkClient) Send(ctx context.Context, in *rusk.SendMessage, opts ...grpc.CallOption) (*rusk.Null, error) {
	res := &rusk.Null{}
	return res, errors.New("not implemented")
}

// Propagate is just here to satisfy the interface. Not used.
func (c *MockNetworkClient) Propagate(ctx context.Context, in *rusk.PropagateMessage, opts ...grpc.CallOption) (*rusk.Null, error) {
	res := &rusk.Null{}
	return res, errors.New("not implemented")
}

// AliveNodes is just here to satisfy the interface. Not used.
func (c *MockNetworkClient) AliveNodes(ctx context.Context, in *rusk.AliveNodesRequest, opts ...grpc.CallOption) (*rusk.AliveNodesResponse, error) {
	res := &rusk.AliveNodesResponse{}
	return res, errors.New("not implemented")
}

//
// Helper functions
//

// createBlockMessage returns a properly encoded wire message
// containing a random block.
func createBlockMessage() (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	// Create the random block and marshal it
	blk := helper.RandomBlock(5525, 10)
	if err := message.MarshalBlock(buf, blk); err != nil {
		return nil, fmt.Errorf("failed to marshal block: %v", err)
	}
	// Add topic to message
	if err := topics.Prepend(buf, topics.Block); err != nil {
		return nil, fmt.Errorf("failed to add topic: %v", err)
	}
	return buf, nil
}

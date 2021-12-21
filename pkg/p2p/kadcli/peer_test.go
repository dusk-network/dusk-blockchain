// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcli

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"testing"

	"google.golang.org/grpc"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/client"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

const (
	RUSK_ADDR = "127.0.0.1:50005"
)

// NetworkServer mocks the Rusk network server
type NetworkServer struct{}

// Listen for messages coming from the network
func (n *NetworkServer) Listen(in *rusk.Null, srv rusk.Network_ListenServer) error {
	//go func() {
	//	for i := 0; i < 10; i++ {
	msg := &rusk.Message{
		Message: []byte("testing"),
		Metadata: &rusk.MessageMetadata{
			KadcastHeight: uint32(1),
			SrcAddress:    RUSK_ADDR,
		},
	}
	if err := srv.Send(msg); err != nil {
		return fmt.Errorf("failed to send msg: %v", err.Error())
	}
	//time.Sleep(100 * time.Millisecond)
	//	}
	//}()
	return nil
}

// Broadcast a message to the network
func (n *NetworkServer) Broadcast(ctx context.Context, msg *rusk.BroadcastMessage) (*rusk.Null, error) {

	return nil, nil
}

// Send a message to a specific target in the network
func (n *NetworkServer) Send(ctx context.Context, msg *rusk.SendMessage) (*rusk.Null, error) {

	return nil, nil
}

// NewRuskMock creates a Rusk Network server mock for tests
func NewRuskMock() (*NetworkServer, error) {

	// create listener
	l, err := net.Listen("tcp", RUSK_ADDR)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	// create grpc server
	s := grpc.NewServer()
	srv := &NetworkServer{}
	rusk.RegisterNetworkServer(s, srv)

	// and start mock service
	if err := s.Serve(l); err != nil {
		return nil, fmt.Errorf("failed to serve: %v", err)
	}

	return srv, nil
}

func TestListen(t *testing.T) {

	t.Log("Starting up!")

	rcvChan := make(chan struct{})

	// Create a reader (+ infra)
	eb := eventbus.New()
	p := peer.NewMessageProcessor(eb)
	g := protocol.NewGossip(protocol.TestNet)

	// Subscribe to Kadcast topic
	respFn := func(srcPeerID string, m message.Message) ([]bytes.Buffer, error) {
		t.Log("got smth!")
		rcvChan <- struct{}{}
		return nil, nil
	}
	p.Register(topics.Kadcast, respFn)

	// Create a rusk mock server
	_, err := NewRuskMock()
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}

	// Connect to rusk (mock)
	_, grpConn := client.CreateNetworkClient(context.Background(), RUSK_ADDR)

	// Create our kadcli (gRPC) Reader
	r := NewReader(eb, g, p, grpConn)
	go r.Listen()

	// Stream something from the server
	<-rcvChan
	t.Log("Done")

}

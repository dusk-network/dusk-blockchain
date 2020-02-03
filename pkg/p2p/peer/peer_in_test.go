package peer

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// Test that the 'ping' message is sent correctly, and that a 'pong' message will result.
func TestPingLoop(t *testing.T) {
	bus := eventbus.New()
	client, srv := net.Pipe()

	exitChan := make(chan struct{}, 1)
	responseChan := make(chan *bytes.Buffer, 10)
	writer := NewWriter(client, processing.NewGossip(protocol.TestNet), bus)
	go writer.Serve(responseChan, exitChan, protocol.FullNode)

	// Set up the other end of the exchange
	responseChan2 := make(chan *bytes.Buffer, 10)
	writer2 := NewWriter(srv, processing.NewGossip(protocol.TestNet), bus)
	go writer2.Serve(responseChan2, exitChan, protocol.FullNode)

	reader := NewReader(client, processing.NewGossip(protocol.TestNet), exitChan)
	go reader.Listen(bus, dupemap.NewDupeMap(0), rpcbus.New(), &chainsync.Counter{}, responseChan, protocol.FullNode, 1*time.Second)

	reader2 := NewReader(srv, processing.NewGossip(protocol.TestNet), exitChan)
	go reader2.Listen(bus, dupemap.NewDupeMap(0), rpcbus.New(), &chainsync.Counter{}, responseChan2, protocol.FullNode, 1*time.Second)

	// We should eventually get a pong message out of responseChan2
	for {
		buf := <-responseChan2
		topic, err := topics.Extract(buf)
		if err != nil {
			t.Fatal(err)
		}

		if topics.Pong.String() == topic.String() {
			break
		}
	}
}

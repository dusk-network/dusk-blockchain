package peer

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/stretchr/testify/assert"
)

// Test that the 'ping' message is sent correctly, and that a 'pong' message will result.
func TestPingLoop(t *testing.T) {
	bus := eventbus.New()
	client, srv := net.Pipe()
	keepAliveTime = 1 * time.Second

	responseChan := make(chan *bytes.Buffer, 10)
	writer := NewWriter(client, processing.NewGossip(protocol.TestNet), bus)
	go writer.Serve(responseChan, make(chan struct{}, 1))

	// Set up the other end of the exchange
	responseChan2 := make(chan *bytes.Buffer, 10)
	writer2 := NewWriter(srv, processing.NewGossip(protocol.TestNet), bus)
	go writer2.Serve(responseChan2, make(chan struct{}, 1))

	reader, err := NewReader(client, processing.NewGossip(protocol.TestNet), dupemap.NewDupeMap(0), bus, rpcbus.New(), &chainsync.Counter{}, responseChan, make(chan struct{}, 1))
	if err != nil {
		t.Fatal(err)
	}
	go reader.ReadLoop()

	reader2, err := NewReader(srv, processing.NewGossip(protocol.TestNet), dupemap.NewDupeMap(0), bus, rpcbus.New(), &chainsync.Counter{}, responseChan2, make(chan struct{}, 1))
	if err != nil {
		t.Fatal(err)
	}
	go reader2.ReadLoop()

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

// TestMalformedFrame ensures peer reader does not panic on
// incomplete checksum buffer
func TestMalformedFrame(t *testing.T) {

	fn := func() {
		peer, r, w, _ := dummyReader()
		defer r.Close()
		defer w.Close()

		// Run the non-recover readLoop to watch for panics
		go peer.readLoop()

		time.Sleep(time.Second)

		// Construct an invalid frame
		frame := new(bytes.Buffer)
		// Add length bytes
		if err := encoding.WriteUint64LE(frame, 7); err != nil {
			t.Error(err)
		}

		// Add correct magic value
		mBuf := protocol.TestNet.ToBuffer()
		if _, err := frame.Write(mBuf.Bytes()); err != nil {
			t.Error(err)
		}

		// Checksum with 3 bytes only
		if _, err := frame.Write([]byte{0, 0, 0}); err != nil {
			t.Error(err)
		}

		_, _ = w.Write(frame.Bytes())
		time.Sleep(1 * time.Second)
	}

	assert.NotPanics(t, fn)
}

func dummyReader() (*Reader, net.Conn, net.Conn, chan<- *bytes.Buffer) {

	bus := eventbus.New()
	rpcbus := rpcbus.New()
	d := dupemap.NewDupeMap(0)
	r, w := net.Pipe()
	keepAliveTime = 30 * time.Second

	respChan := make(chan *bytes.Buffer, 10)
	g := processing.NewGossip(protocol.TestNet)
	peer, _ := NewReader(r, g, d, bus, rpcbus, &chainsync.Counter{},
		respChan, make(chan struct{}, 1))

	return peer, r, w, respChan
}

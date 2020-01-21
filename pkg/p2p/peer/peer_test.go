package peer_test

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/stretchr/testify/assert"
)

var receiveFn = func(c net.Conn) {
	for {
		c.SetReadDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 1024)
		if _, err := c.Read(buf); err != nil {
			break
		}
	}
}

// Test the functionality of the peer.Reader through the ReadLoop.
func TestReader(t *testing.T) {
	g := processing.NewGossip(protocol.TestNet)
	client, srv := net.Pipe()
	go func() {
		buf := makeAgreementBuffer(10)
		err := g.Process(buf)
		if err != nil {
			t.Fatal(err)
		}
		client.Write(buf.Bytes())
	}()

	eb := eventbus.New()
	rpcBus := rpcbus.New()
	peerReader := helper.StartPeerReader(srv)

	// Our message should come in on the agreement topic
	agreementChan := make(chan bytes.Buffer, 1)
	l := eventbus.NewChanListener(agreementChan)
	eb.Subscribe(topics.Agreement, l)

	go peerReader.Listen(eb, dupemap.NewDupeMap(0), rpcBus, chainsync.NewCounter(eb), nil, protocol.FullNode)

	// We should get the message through this channel
	<-agreementChan
}

// Test the functionality of the peer.Writer through the use of the ring buffer.
func TestWriteRingBuffer(t *testing.T) {
	bus := eventbus.New()

	for i := 0; i < 100; i++ {
		p := addPeer(bus, receiveFn)
		defer p.Conn.Close()
	}

	ev := makeAgreementBuffer(10)
	if err := topics.Prepend(ev, topics.Agreement); err != nil {
		panic(err)
	}

	for i := 0; i < 1000; i++ {
		bus.Publish(topics.Gossip, ev)
	}
}

// Test the functionality of the peer.Writer through the use of the outgoing message queue.
func TestWriteLoop(t *testing.T) {
	bus := eventbus.New()
	client, srv := net.Pipe()

	g := processing.NewGossip(protocol.TestNet)
	buf := makeAgreementBuffer(10)
	go func(g *processing.Gossip) {
		responseChan := make(chan *bytes.Buffer)
		writer := peer.NewWriter(client, g, bus)
		go writer.Serve(responseChan, make(chan struct{}, 1), protocol.FullNode)

		bufCopy := *buf
		responseChan <- &bufCopy
	}(g)

	// Decode and remove magic
	length, err := g.UnpackLength(srv)
	assert.NoError(t, err)

	decoded := make([]byte, length)
	_, err = io.ReadFull(srv, decoded)
	assert.NoError(t, err)

	// Remove checksum
	decoded = decoded[4:]

	assert.Equal(t, decoded, buf.Bytes())
}

// Test that the 'ping' message is sent correctly, and that a 'pong' message will result.
func TestPingLoop(t *testing.T) {
	bus := eventbus.New()
	client, srv := net.Pipe()

	responseChan := make(chan *bytes.Buffer, 10)
	writer := peer.NewWriter(client, processing.NewGossip(protocol.TestNet), bus)
	go writer.Serve(responseChan, make(chan struct{}, 1), protocol.FullNode)

	// Set up the other end of the exchange
	responseChan2 := make(chan *bytes.Buffer, 10)
	writer2 := peer.NewWriter(srv, processing.NewGossip(protocol.TestNet), bus)
	go writer2.Serve(responseChan2, make(chan struct{}, 1), protocol.FullNode)
	// Give the goroutine some time to start
	time.Sleep(100 * time.Millisecond)

	reader := peer.NewReader(client, processing.NewGossip(protocol.TestNet), make(chan struct{}, 1))
	go reader.Listen(bus, dupemap.NewDupeMap(0), rpcbus.New(), &chainsync.Counter{}, responseChan2, protocol.FullNode)

	// We should eventually get a pong message out of responseChan2
	buf := <-responseChan2
	topic, err := topics.Extract(buf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, topics.Pong.String(), topic.String())
}

// Test that peers with a LightNode service flag only get specific
// messages, and can only send specific messages.
func TestServiceFlagGuard(t *testing.T) {
	bus := eventbus.New()
	client, srv := net.Pipe()
	agChan := make(chan bytes.Buffer, 1)
	bus.Subscribe(topics.Agreement, eventbus.NewChanListener(agChan))

	responseChan := make(chan *bytes.Buffer, 10)
	writer := peer.NewWriter(client, processing.NewGossip(protocol.TestNet), bus)
	go writer.Serve(responseChan, make(chan struct{}, 1), protocol.LightNode)

	// Give the goroutine some time to start
	time.Sleep(100 * time.Millisecond)

	reader := peer.NewReader(client, processing.NewGossip(protocol.TestNet), make(chan struct{}, 1))
	go reader.Listen(bus, dupemap.NewDupeMap(0), rpcbus.New(), &chainsync.Counter{}, responseChan, protocol.LightNode)

	// Send an agreement buffer from the peer. This should not be routed
	g := processing.NewGossip(protocol.TestNet)
	buf := makeAgreementBuffer(15)
	assert.NoError(t, g.Process(buf))

	_, err := srv.Write(buf.Bytes())
	assert.NoError(t, err)

	// Should not get anything on the agChan
	select {
	case <-agChan:
		t.Fatal("should not have gotten any message on agChan")
	case <-time.After(1 * time.Second):
		// Success
	}

	// Now, attempt to send an agreement buffer to the peer. It should
	// never arrive.
	buf = makeAgreementBuffer(15)
	bus.Publish(topics.Gossip, buf)

	srv.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = srv.Read([]byte{0})
	assert.Error(t, err)
}

func BenchmarkWriter(b *testing.B) {
	bus := eventbus.New()

	for i := 0; i < 100; i++ {
		p := addPeer(bus, receiveFn)
		defer p.Conn.Close()
	}

	ev := makeAgreementBuffer(10)
	if err := topics.Prepend(ev, topics.Agreement); err != nil {
		panic(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(topics.Gossip, ev)
	}
}

func makeAgreementBuffer(keyAmount int) *bytes.Buffer {
	p, keys := consensus.MockProvisioners(keyAmount)

	buf := agreement.MockAgreement(make([]byte, 32), 1, 1, keys, p)
	if err := topics.Prepend(buf, topics.Agreement); err != nil {
		panic(err)
	}

	return buf
}

func addPeer(bus *eventbus.EventBus, receiveFunc func(net.Conn)) *peer.Writer {
	client, srv := net.Pipe()
	g := processing.NewGossip(protocol.TestNet)
	pw := peer.NewWriter(client, g, bus)
	go receiveFunc(srv)
	return pw
}

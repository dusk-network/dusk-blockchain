package peer_test

import (
	"bufio"
	"bytes"
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
	peerReader, err := helper.StartPeerReader(srv, eb, rpcBus, chainsync.NewCounter(eb), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Our message should come in on the agreement topic
	agreementChan := make(chan bytes.Buffer, 1)
	l := eventbus.NewChanListener(agreementChan)
	eb.Subscribe(topics.Agreement, l)

	go peerReader.ReadLoop()

	// We should get the message through this channel
	<-agreementChan
}

// Test the functionality of the peer.Writer through the use of the ring buffer.
func TestWriteRingBuffer(t *testing.T) {
	bus := eventbus.New()
	g := processing.NewGossip(protocol.TestNet)
	bus.Register(topics.Gossip, g)

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

	buf := makeAgreementBuffer(10)
	go func() {
		responseChan := make(chan *bytes.Buffer)
		g := processing.NewGossip(protocol.TestNet)
		writer := peer.NewWriter(client, g, bus)
		go writer.Serve(responseChan, make(chan struct{}, 1))

		bufCopy := *buf
		responseChan <- &bufCopy
	}()

	r := bufio.NewReader(srv)
	length, err := processing.ReadFrame(r)
	if err != nil {
		t.Fatal(err)
	}

	// Decode and remove magic
	bs := make([]byte, length)
	_, err = r.Read(bs)

	if !assert.NoError(t, err) {
		t.FailNow()
	}

	decoded := bytes.NewBuffer(bs)
	assert.Equal(t, decoded.Bytes()[4:], buf.Bytes())
}

// Test that the 'ping' message is sent correctly, and that a 'pong' message will result.
func TestPingLoop(t *testing.T) {
	bus := eventbus.New()
	client, srv := net.Pipe()

	responseChan := make(chan *bytes.Buffer, 10)
	writer := peer.NewWriter(client, processing.NewGossip(protocol.TestNet), bus)
	go writer.Serve(responseChan, make(chan struct{}, 1))

	// Set up the other end of the exchange
	responseChan2 := make(chan *bytes.Buffer, 10)
	writer2 := peer.NewWriter(srv, processing.NewGossip(protocol.TestNet), bus)
	go writer2.Serve(responseChan2, make(chan struct{}, 1))
	// Give the goroutine some time to start
	time.Sleep(100 * time.Millisecond)

	reader, err := peer.NewReader(client, processing.NewGossip(protocol.TestNet), dupemap.NewDupeMap(0), bus, rpcbus.New(), &chainsync.Counter{}, responseChan2, make(chan struct{}, 1))
	if err != nil {
		t.Fatal(err)
	}
	go reader.ReadLoop()
	// Read the `mempool` message
	<-responseChan2

	// We should eventually get a pong message out of responseChan2
	buf := <-responseChan2
	topic, err := topics.Extract(buf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, topics.Pong.String(), topic.String())
}

func BenchmarkWriter(b *testing.B) {
	bus := eventbus.New()
	g := processing.NewGossip(protocol.TestNet)
	bus.Register(topics.Gossip, g)

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

	buf := agreement.MockAgreement(make([]byte, 32), 1, 2, keys, p.CreateVotingCommittee(1, 2, keyAmount))
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

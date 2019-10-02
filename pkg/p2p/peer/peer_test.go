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
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
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
	eb.Subscribe(string(topics.Agreement), l)

	go peerReader.ReadLoop()

	// We should get the message through this channel
	<-agreementChan
}

// Test the functionality of the peer.Writer through the use of the ring buffer.
func TestWriteRingBuffer(t *testing.T) {
	bus := eventbus.New()
	g := processing.NewGossip(protocol.TestNet)
	bus.Register(string(topics.Gossip), g)

	for i := 0; i < 100; i++ {
		p := addPeer(bus, receiveFn)
		defer p.Conn.Close()
	}

	ev := makeAgreementBuffer(10)
	msg, err := wire.AddTopic(ev, topics.Agreement)
	if err != nil {
		panic(err)
	}

	for i := 0; i < 1000; i++ {
		bus.Publish(string(topics.Gossip), msg)
	}
}

// Test the functionality of the peer.Writer through the use of the outgoing message queue.
func TestWriteLoop(t *testing.T) {
	bus := eventbus.New()
	client, srv := net.Pipe()

	buf := makeAgreementBuffer(10)
	go func() {
		responseChan := make(chan *bytes.Buffer)
		writer := peer.NewWriter(client, protocol.TestNet, bus)
		go writer.Serve(responseChan, make(chan struct{}, 1))

		bufCopy := *buf
		responseChan <- &bufCopy
	}()

	r := bufio.NewReader(srv)
	bs, err := processing.ReadFrame(r)
	if err != nil {
		t.Fatal(err)
	}

	// Decode and remove magic
	decoded := bytes.NewBuffer(bs)

	assert.Equal(t, decoded.Bytes()[4:], buf.Bytes())
}

func BenchmarkWriter(b *testing.B) {
	bus := eventbus.New()
	g := processing.NewGossip(protocol.TestNet)
	bus.Register(string(topics.Gossip), g)

	for i := 0; i < 100; i++ {
		p := addPeer(bus, receiveFn)
		defer p.Conn.Close()
	}

	ev := makeAgreementBuffer(10)
	msg, err := wire.AddTopic(ev, topics.Agreement)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(string(topics.Gossip), msg)
	}
}

func makeAgreementBuffer(keyAmount int) *bytes.Buffer {
	p, keys := consensus.MockProvisioners(keyAmount)

	buf := agreement.MockAgreement(make([]byte, 32), 1, 2, keys, p.CreateVotingCommittee(1, 2, keyAmount))
	withTopic, err := wire.AddTopic(buf, topics.Agreement)
	if err != nil {
		panic(err)
	}

	return withTopic
}

func addPeer(bus *eventbus.EventBus, receiveFunc func(net.Conn)) *peer.Writer {
	client, srv := net.Pipe()
	pw := peer.NewWriter(client, protocol.TestNet, bus)
	go receiveFunc(srv)
	return pw
}

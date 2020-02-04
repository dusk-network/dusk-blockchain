package peer_test

import (
	"bytes"
	"encoding/hex"
	"io"
	"net"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
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
		msg := makeAgreementGossip(10)
		buf, err := message.Marshal(msg)
		if err != nil {
			t.Fatal(err)
		}
		if err := g.Process(&buf); err != nil {
			t.Fatal(err)
		}
		client.Write(buf.Bytes())
	}()

	eb := eventbus.New()
	rpcBus := rpcbus.New()
	peerReader := helper.StartPeerReader(srv)

	// Our message should come in on the agreement topic
	agreementChan := make(chan message.Message, 1)
	l := eventbus.NewChanListener(agreementChan)
	eb.Subscribe(topics.Agreement, l)

	go peerReader.Listen(eb, dupemap.NewDupeMap(0), rpcBus, chainsync.NewCounter(eb), nil, protocol.FullNode, 30*time.Second)

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

	ev := makeAgreementGossip(10)
	msg := message.New(topics.Agreement, ev)

	for i := 0; i < 1000; i++ {
		bus.Publish(topics.Gossip, msg)
	}
}

// Test the functionality of the peer.Writer through the use of the outgoing message queue.
func TestWriteLoop(t *testing.T) {
	bus := eventbus.New()
	client, srv := net.Pipe()

	g := processing.NewGossip(protocol.TestNet)
	msg := makeAgreementGossip(10)
	buf, err := message.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	go func(g *processing.Gossip) {
		responseChan := make(chan *bytes.Buffer)
		writer := peer.NewWriter(client, g, bus, 30*time.Millisecond)
		go writer.Serve(responseChan, make(chan struct{}, 1), protocol.FullNode)

		bufCopy := buf
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

	assert.Equal(t, decoded, (&buf).Bytes())
}

// Test that peers with a LightNode service flag only get specific
// messages, and can only send specific messages.
func TestServiceFlagGuard(t *testing.T) {
	bus := eventbus.New()
	client, srv := net.Pipe()
	agChan := make(chan message.Message, 1)
	bus.Subscribe(topics.Agreement, eventbus.NewChanListener(agChan))

	responseChan := make(chan *bytes.Buffer, 10)
	writer := peer.NewWriter(client, processing.NewGossip(protocol.TestNet), bus)
	go writer.Serve(responseChan, make(chan struct{}, 1), protocol.LightNode)

	reader := peer.NewReader(client, processing.NewGossip(protocol.TestNet), make(chan struct{}, 1))
	go reader.Listen(bus, dupemap.NewDupeMap(0), rpcbus.New(), &chainsync.Counter{}, responseChan, protocol.LightNode, 30*time.Second)

	// Send an agreement buffer from the peer. This should not be routed
	g := processing.NewGossip(protocol.TestNet)
	msg := makeAgreementGossip(15)
	buf, err := message.Marshal(msg)
	assert.NoError(t, g.Process(&buf))

	_, err = srv.Write(buf.Bytes())
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
	msg = makeAgreementGossip(15)
	buf, err = message.Marshal(msg)
	msg = message.New(topics.Agreement, buf)
	bus.Publish(topics.Gossip, msg)

	srv.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = srv.Read([]byte{0})
	assert.Error(t, err)
}

// Test that, when in light-node mode, all functionality works as
// expected.
func TestLightNode(t *testing.T) {
	// Mock config to go into light node mode
	r := new(config.Registry)
	r.Database.Driver = "lite_v0.1.0"
	r.General.Network = "testnet"
	r.General.LightNode = true
	r.Mempool.MaxInvItems = 10000
	config.Mock(r)

	bus, rb := eventbus.New(), rpcbus.New()
	respondToGetLastBlock(rb)
	client, srv := net.Pipe()

	// Channels for monitoring the event bus
	agChan := make(chan message.Message, 1)
	bus.Subscribe(topics.Agreement, eventbus.NewChanListener(agChan))
	blkChan := make(chan message.Message, 1)
	bus.Subscribe(topics.Block, eventbus.NewChanListener(blkChan))

	// Setting up the peer
	responseChan := make(chan *bytes.Buffer, 10)
	reader := peer.NewReader(client, processing.NewGossip(protocol.TestNet), make(chan struct{}, 1))
	go reader.Listen(bus, dupemap.NewDupeMap(0), rb, &chainsync.Counter{}, responseChan, protocol.FullNode, 30*time.Second)

	// Should not attempt to send a `mempool` message
	select {
	case <-responseChan:
		t.Fatal("should not have gotten anything on the responseChan")
	case <-time.After(1 * time.Second):
		// success
	}

	// Should not send a `getdata` on a tx `inv`
	g := processing.NewGossip(protocol.TestNet)
	msg := &peermsg.Inv{}
	hash, _ := crypto.RandEntropy(32)
	msg.AddItem(peermsg.InvTypeMempoolTx, hash)
	buf := new(bytes.Buffer)
	assert.NoError(t, msg.Encode(buf))
	assert.NoError(t, topics.Prepend(buf, topics.Inv))
	assert.NoError(t, g.Process(buf))

	_, err := srv.Write(buf.Bytes())
	assert.NoError(t, err)

	select {
	case <-responseChan:
		t.Fatal("should not have gotten anything on the responseChan")
	case <-time.After(1 * time.Second):
		// success
	}

	// Should not respond to consensus messages
	agMsg := makeAgreementGossip(15)
	agBuf, err := message.Marshal(agMsg)
	assert.NoError(t, g.Process(&agBuf))

	_, err = srv.Write(buf.Bytes())
	assert.NoError(t, err)

	select {
	case <-agChan:
		t.Fatal("should not have gotten anything on the agChan")
	case <-time.After(1 * time.Second):
		// success
	}

	// Should be able to ask for and receive blocks
	msg = &peermsg.Inv{}
	hash, _ = crypto.RandEntropy(32)
	msg.AddItem(peermsg.InvTypeBlock, hash)
	buf = new(bytes.Buffer)
	assert.NoError(t, msg.Encode(buf))
	assert.NoError(t, topics.Prepend(buf, topics.Inv))
	assert.NoError(t, g.Process(buf))

	_, err = srv.Write(buf.Bytes())
	assert.NoError(t, err)

	// Should get a `getdata` on the responseChan
	select {
	case m := <-responseChan:
		assert.Equal(t, topics.GetData, topics.Topic(m.Bytes()[0]))
	case <-time.After(1 * time.Second):
		t.Fatal("should have gotten a getdata message on the responseChan")
	}

	// Let's give a block back
	blk := helper.RandomBlock(t, 1, 1)
	buf = new(bytes.Buffer)
	assert.NoError(t, message.MarshalBlock(buf, blk))
	assert.NoError(t, topics.Prepend(buf, topics.Block))
	assert.NoError(t, g.Process(buf))

	_, err = srv.Write(buf.Bytes())
	assert.NoError(t, err)

	// Should be broadcast on the event bus
	select {
	case <-blkChan:
		// success
	case <-time.After(1 * time.Second):
		t.Fatal("should have gotten a block on the event bus")
	}
}

func BenchmarkWriter(b *testing.B) {
	bus := eventbus.New()

	for i := 0; i < 100; i++ {
		p := addPeer(bus, receiveFn)
		defer p.Conn.Close()
	}

	msg := makeAgreementGossip(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(topics.Gossip, msg)
	}
}

func respondToGetLastBlock(rb *rpcbus.RPCBus) {
	c := make(chan rpcbus.Request, 1)
	rb.Register(rpcbus.GetLastBlock, c)

	go func(c chan rpcbus.Request) {
		r := <-c

		blob, err := hex.DecodeString(config.TestNetGenesisBlob)
		if err != nil {
			panic(err)
		}

		r.RespChan <- rpcbus.Response{*bytes.NewBuffer(blob), nil}
	}(c)
}

func makeAgreementGossip(keyAmount int) message.Message {
	p, keys := consensus.MockProvisioners(keyAmount)
	aggro := message.MockAgreement(make([]byte, 32), 1, 1, keys, p)
	return message.New(topics.Agreement, aggro)
}

func addPeer(bus *eventbus.EventBus, receiveFunc func(net.Conn)) *peer.Writer {
	client, srv := net.Pipe()
	g := processing.NewGossip(protocol.TestNet)
	pw := peer.NewWriter(client, g, bus)
	go receiveFunc(srv)
	return pw
}

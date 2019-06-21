package peer_test

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing/chainsync"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
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

// TODO: this test should be expanded upon, as it doesn't test that much right now
func TestReader(t *testing.T) {
	client, srv := net.Pipe()
	test := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	go func() {
		for i := 0; i < 10000; i++ {
			client.Write(test)
		}

		client.Write([]byte{0})
		client.Close()
	}()

	eb := wire.NewEventBus()
	rpcBus := wire.NewRPCBus()
	peerReader, err := helper.StartPeerReader(srv, eb, rpcBus, chainsync.NewCounter(eb), nil)
	if err != nil {
		t.Fatal(err)
	}
	// This should block until the connection is closed, which should happen after
	// two and a half seconds.
	peerReader.ReadLoop()
}

func TestWriter(t *testing.T) {
	bus := wire.NewEventBus()
	g := processing.NewGossip(protocol.TestNet)
	bus.RegisterPreprocessor(string(topics.Gossip), g)

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
		bus.Stream(string(topics.Gossip), msg)
	}
}

func BenchmarkWriter(b *testing.B) {
	bus := wire.NewEventBus()
	g := processing.NewGossip(protocol.TestNet)
	bus.RegisterPreprocessor(string(topics.Gossip), g)

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
		bus.Stream(string(topics.Gossip), msg)
	}
}

func makeAgreementBuffer(keyAmount int) *bytes.Buffer {
	var keys []user.Keys
	for i := 0; i < keyAmount; i++ {
		keyPair, _ := user.NewRandKeys()
		keys = append(keys, keyPair)
	}

	return agreement.MockAgreement(make([]byte, 32), 1, 2, keys)
}

func addPeer(bus *wire.EventBus, receiveFunc func(net.Conn)) *peer.Writer {
	client, srv := net.Pipe()
	pw := peer.NewWriter(client, protocol.TestNet, bus)
	pw.Subscribe(bus)
	go receiveFunc(srv)
	return pw
}

type mockCollector struct {
	t *testing.T
}

func (m *mockCollector) Collect(b *bytes.Buffer) error {
	assert.NotEmpty(m.t, b)
	return nil
}

type mockSynchronizer struct {
}

func (m *mockSynchronizer) Synchronize(conn net.Conn, blockChan <-chan *bytes.Buffer) {
	return
}

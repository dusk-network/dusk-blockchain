package peer_test

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/dupemap"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func TestScanner(t *testing.T) {
	test := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	go func() {
		conn, err := net.Dial("tcp", ":3000")
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		for i := 0; i < 10000; i++ {
			conn.Write(test)
		}

		conn.Write([]byte{0})
	}()

	l, err := net.Listen("tcp", ":3000")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	conn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	dupeMap := dupemap.NewDupeMap(5)
	eb := wire.NewEventBus()
	peerReader, err := peer.NewReader(conn, protocol.TestNet, dupeMap, eb, &mockSynchronizer{})
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

	receiveFn := func(c net.Conn, doneChan chan struct{}, outboundChan chan struct{}) {
		for {
			c.SetReadDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 1024)
			if _, err := c.Read(buf); err != nil {
				break
			}
		}
	}

	l, err := startServer(receiveFn, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	for i := 0; i < 100; i++ {
		p := addPeer(bus)
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

	receiveFn := func(c net.Conn, doneChan chan struct{}, outboundChan chan struct{}) {
		for {
			c.SetReadDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 1024)
			if _, err := c.Read(buf); err != nil {
				break
			}
		}
	}

	l, err := startServer(receiveFn, nil, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()
	for i := 0; i < 100; i++ {
		p := addPeer(bus)
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

func startServer(f func(net.Conn, chan struct{}, chan struct{}), inboundChan chan struct{},
	outboundChan chan struct{}) (net.Listener, error) {
	l, err := net.Listen("tcp", ":3000")
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}

			go f(conn, inboundChan, outboundChan)
		}
	}()
	return l, nil
}

func makeAgreementBuffer(keyAmount int) *bytes.Buffer {
	var keys []user.Keys
	for i := 0; i < keyAmount; i++ {
		keyPair, _ := user.NewRandKeys()
		keys = append(keys, keyPair)
	}

	return agreement.MockAgreement(make([]byte, 32), 1, 2, keys)
}

func addPeer(bus *wire.EventBus) *peer.Writer {
	conn, err := net.Dial("tcp", ":3000")
	if err != nil {
		panic(err)
	}

	pw := peer.NewWriter(conn, protocol.TestNet, bus)
	pw.Subscribe(bus)
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

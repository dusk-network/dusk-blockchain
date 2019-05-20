package peer

import (
	"bytes"
	"net"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"github.com/stretchr/testify/assert"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

var empty struct{}

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

	peerReader := NewReader(conn, protocol.TestNet)
	collector := &mockCollector{t}

	// This should block until the connection is closed, which should happen after
	// two and a half seconds.
	peerReader.ReadLoop(collector)
}

func TestWriter(t *testing.T) {
	bus := wire.NewEventBus()
	g := NewGossip(protocol.TestNet)
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

	startServer(receiveFn, nil, nil)
	for i := 0; i < 100; i++ {
		addPeer(bus)
	}

	ev := agreement.MockAgreementBuf(make([]byte, 32), 1, 2, 10)
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
	g := NewGossip(protocol.TestNet)
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

	startServer(receiveFn, nil, nil)
	for i := 0; i < 100; i++ {
		addPeer(bus)
	}

	ev := agreement.MockAgreementBuf(make([]byte, 32), 1, 2, 10)
	msg, err := wire.AddTopic(ev, topics.Agreement)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Stream(string(topics.Gossip), msg)
	}
	b.StopTimer()
}

func BenchmarkFastReceiver(b *testing.B) {
	log.SetLevel(log.ErrorLevel)
	// cpuFile, err := os.Create("cpu.prof")
	// if err != nil {
	// 	b.Fatal(err)
	// }
	// defer cpuFile.Close()

	// if err := pprof.StartCPUProfile(cpuFile); err != nil {
	// 	b.Fatal(err)
	// }

	bus := wire.NewEventBus()

	receiveFn := func(c net.Conn, doneChan chan struct{}, outboundChan chan struct{}) {
		for {
			c.SetReadDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 1024)
			if _, err := c.Read(buf); err != nil {
				break
			}
		}

	}

	startServer(receiveFn, nil, nil)
	for i := 0; i < 8; i++ {
		addPeer(bus)
	}

	ev := agreement.MockAgreementBuf(make([]byte, 32), 1, 2, 10)
	msg, err := wire.AddTopic(ev, topics.Agreement)
	if err != nil {
		panic(err)
	}
	// b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// b.StartTimer()
		copy := *msg
		bus.Stream(string(topics.Gossip), &copy)
		// b.StopTimer()
	}
	// pprof.StopCPUProfile()
}

func BenchmarkFastEventReceiver(b *testing.B) {
	log.SetLevel(log.ErrorLevel)
	// cpuFile, err := os.Create("cpu.prof")
	// if err != nil {
	// 	b.Fatal(err)
	// }
	// defer cpuFile.Close()

	// if err := pprof.StartCPUProfile(cpuFile); err != nil {
	// 	b.Fatal(err)
	// }

	bus := wire.NewEventBus()

	receiveFn := func(c net.Conn, inboundChan chan struct{}, outboundChan chan struct{}) {
		for {
			c.SetReadDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 1024)
			if _, err := c.Read(buf); err != nil {
				outboundChan <- empty
				break
			}
		}
	}

	inboundChan := make(chan struct{}, 10)
	go startServer(receiveFn, nil, inboundChan)
	for i := 0; i < 100; i++ {
		addEventPeer(bus, b)
	}

	ev := agreement.MockAgreementBuf(make([]byte, 32), 1, 2, 10)
	msg, err := wire.AddTopic(ev, topics.Agreement)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy := msg
		bus.Stream(string(topics.Gossip), copy)
	}
	<-inboundChan
	// pprof.StopCPUProfile()
}

func startServer(f func(net.Conn, chan struct{}, chan struct{}), inboundChan chan struct{},
	outboundChan chan struct{}) error {
	l, err := net.Listen("tcp", ":3000")
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				panic(err)
			}

			go f(conn, inboundChan, outboundChan)
		}
	}()
	return nil
}

func addPeer(bus *wire.EventBus) {
	conn, err := net.Dial("tcp", ":3000")
	if err != nil {
		panic(err)
	}

	pw := NewWriter(conn, protocol.TestNet, bus)
	pw.Subscribe()
}

func addEventPeer(bus *wire.EventBus, b *testing.B) {
	conn, err := net.Dial("tcp", ":3000")
	if err != nil {
		panic(err)
	}

	newPeerEventWriter(conn, bus)
}

type peerEventWriter struct {
	net.Conn
	gossipChan <-chan *bytes.Buffer
}

func newPeerEventWriter(conn net.Conn, bus *wire.EventBus) {
	gossipChan := make(chan *bytes.Buffer, 100)
	bus.Subscribe(string(topics.Gossip), gossipChan)
	pew := &peerEventWriter{conn, gossipChan}
	go pew.writeLoop()
}

func (p *peerEventWriter) writeLoop() {
	for {
		buf := <-p.gossipChan
		p.Write(buf.Bytes())
	}
}

type mockCollector struct {
	t *testing.T
}

func (m *mockCollector) Collect(b *bytes.Buffer) error {
	assert.NotEmpty(m.t, b)
	return nil
}

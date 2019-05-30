package peer_test

import (
	"net"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

func TestHandshake(t *testing.T) {
	eb := wire.NewEventBus()
	go func() {
		pr := startReader()
		if err := pr.Handshake(); err != nil {
			t.Fatal(err)
		}
	}()

	// allow some time for the reader to start listening
	time.Sleep(time.Millisecond * 100)

	pw := startWriter(eb)
	if err := pw.Handshake(); err != nil {
		t.Fatal(err)
	}
}

func startReader() *peer.Reader {
	l, err := net.Listen("tcp", ":3000")
	if err != nil {
		panic(err)
	}

	conn, err := l.Accept()
	if err != nil {
		panic(err)
	}

	return peer.NewReader(conn, protocol.TestNet)
}

func startWriter(subscriber wire.EventSubscriber) *peer.Writer {
	conn, err := net.Dial("tcp", ":3000")
	if err != nil {
		panic(err)
	}

	return peer.NewWriter(conn, protocol.TestNet, subscriber)
}

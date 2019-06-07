package peer_test

import (
	"net"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

var port = "3000"

func TestHandshake(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	eb := wire.NewEventBus()
	go func() {
		pr, err := helper.StartPeerReader(eb, &mockSynchronizer{}, port)
		if err != nil {
			t.Fatal(err)
		}
		if err := pr.Handshake(); err != nil {
			t.Fatal(err)
		}
	}()

	// allow some time for the reader to start listening
	time.Sleep(time.Millisecond * 100)

	pw := startWriter(eb)
	defer pw.Conn.Close()
	if err := pw.Handshake(); err != nil {
		t.Fatal(err)
	}
}

func startWriter(subscriber wire.EventSubscriber) *peer.Writer {
	conn, err := net.Dial("tcp", ":"+port)
	if err != nil {
		panic(err)
	}

	return peer.NewWriter(conn, protocol.TestNet, subscriber)
}

package peer_test

import (
	"net"
	"testing"
	"time"

	_ "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/lite"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing/chainsync"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

func TestHandshake(t *testing.T) {

	eb := wire.NewEventBus()
	rpcBus := wire.NewRPCBus()
	counter := chainsync.NewCounter(eb)
	client, srv := net.Pipe()

	go func() {
		peerReader, err := helper.StartPeerReader(srv, eb, rpcBus, counter, nil)
		if err != nil {
			t.Fatal(err)
		}

		if err := peerReader.Accept(); err != nil {
			t.Fatal(err)
		}
	}()

	time.Sleep(500 * time.Millisecond)
	pw := peer.NewWriter(client, protocol.TestNet, eb)
	defer pw.Conn.Close()
	if err := pw.Handshake(); err != nil {
		t.Fatal(err)
	}
}

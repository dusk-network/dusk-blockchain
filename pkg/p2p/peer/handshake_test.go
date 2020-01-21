package peer_test

import (
	"net"
	"testing"
	"time"

	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

func TestHandshake(t *testing.T) {

	eb := eventbus.New()
	client, srv := net.Pipe()

	go func() {
		peerReader := helper.StartPeerReader(srv)

		if _, err := peerReader.Accept(); err != nil {
			t.Fatal(err)
		}
	}()

	time.Sleep(500 * time.Millisecond)
	g := processing.NewGossip(protocol.TestNet)
	pw := peer.NewWriter(client, g, eb)
	defer pw.Conn.Close()
	if _, err := pw.Handshake(); err != nil {
		t.Fatal(err)
	}
}

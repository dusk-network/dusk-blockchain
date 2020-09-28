package peer

import (
	"net"
	"testing"
	"time"

	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

func TestHandshake(t *testing.T) {

	eb := eventbus.New()
	rpcBus := rpcbus.New()
	counter := chainsync.NewCounter()

	client, srv := net.Pipe()

	go func() {
		peerReader, err := StartPeerReader(srv, eb, rpcBus, counter, nil)
		if err != nil {
			panic(err)
		}

		if err := peerReader.Accept(); err != nil {
			panic(err)
		}
	}()

	time.Sleep(500 * time.Millisecond)
	g := processing.NewGossip(protocol.TestNet)
	pw := NewWriter(client, g, eb)
	defer func() {
		_ = pw.Conn.Close()
	}()
	if err := pw.Handshake(); err != nil {
		t.Fatal(err)
	}
}

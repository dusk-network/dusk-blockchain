package peer

import (
	"net"
	"os"
	"testing"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/stretchr/testify/require"

	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

func TestHandshake(t *testing.T) {

	//setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := cfg.LoadFromFile(cwd + "/../../../dusk.toml")
	require.Nil(t, err)
	cfg.Mock(&r)

	eb := eventbus.New()
	rpcBus := rpcbus.New()
	counter := chainsync.NewCounter()

	client, srv := net.Pipe()

	go func() {
		peerReader := StartPeerReader(srv, eb, rpcBus, counter, nil)

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

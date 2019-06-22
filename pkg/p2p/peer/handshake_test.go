package peer_test

import (
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing/chainsync"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

func mockConfig(t *testing.T) func() {
	storeDir, err := ioutil.TempDir(os.TempDir(), "peer_test")
	if err != nil {
		t.Fatal(err.Error())
	}

	r := cfg.Registry{}
	r.Database.Dir = storeDir
	r.Database.Driver = heavy.DriverName
	r.General.Network = "testnet"
	cfg.Mock(&r)

	return func() {
		os.RemoveAll(storeDir)
	}
}

func TestHandshake(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

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

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
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

var port = "3000"

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

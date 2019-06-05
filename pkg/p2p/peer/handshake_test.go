package peer_test

import (
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/dupemap"
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
	go func() {
		pr := startReader(eb)
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

func startReader(bus *wire.EventBus) *peer.Reader {
	l, err := net.Listen("tcp", ":3000")
	if err != nil {
		panic(err)
	}

	conn, err := l.Accept()
	if err != nil {
		panic(err)
	}

	dupeMap := dupemap.NewDupeMap(5)
	pw, err := peer.NewReader(conn, protocol.TestNet, dupeMap, bus, &mockSynchronizer{})
	if err != nil {
		panic(err)
	}

	return pw
}

func startWriter(subscriber wire.EventSubscriber) *peer.Writer {
	conn, err := net.Dial("tcp", ":3000")
	if err != nil {
		panic(err)
	}

	return peer.NewWriter(conn, protocol.TestNet, subscriber)
}

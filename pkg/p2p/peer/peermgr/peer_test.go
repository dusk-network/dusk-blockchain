package peermgr_test

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

func createResponseHandler() peermgr.ResponseHandler {

	//DefaultHeight := func() uint64 {
	//	return 10
	//}

	OnAddr := func(p *peermgr.Peer, msg *payload.MsgAddr) {}
	OnHeaders := func(p *peermgr.Peer, msg *payload.MsgHeaders) {}
	OnGetHeaders := func(p *peermgr.Peer, msg *payload.MsgGetHeaders) {}
	//OnInv := func(p *peermgr.Peer, msg *payload.MsgInv) {}
	OnGetData := func(p *peermgr.Peer, msg *payload.MsgGetData) {}
	//OnBlock := func(p *peermgr.Peer, msg *payload.MsgBlock) {}
	OnGetBlocks := func(p *peermgr.Peer, msg *payload.MsgGetBlocks) {}

	return peermgr.ResponseHandler{
		OnHeaders:    OnHeaders,
		OnAddr:       OnAddr,
		OnGetHeaders: OnGetHeaders,
		//OnInv:        OnInv,
		OnGetData: OnGetData,
		//OnBlock:      OnBlock,
		OnGetBlocks: OnGetBlocks,
	}
}

// TODO: Test is an integration test, not unit test.
// Let's find an integration test lib or mocking.
func TestHandshake(t *testing.T) {
	address := ":20338"
	viper.Set("net.magic", "1953721920")
	go func() {

		conn, err := net.DialTimeout("tcp", address, 200*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		rspHndlr := createResponseHandler()
		p := peermgr.NewPeer(conn, true, rspHndlr)
		err = p.Run()
		verack := payload.NewMsgVerAck()
		if err != nil {
			t.Fail()
		}
		if err := p.Write(verack); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, true, p.IsVerackReceived())

	}()

	listener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatal(err)
		return
	}

	defer func() {
		listener.Close()
	}()

	for {

		conn, err := listener.Accept()
		if err != nil {
			t.Fatal(err)
		}

		fromAddr := payload.NewNetAddress("82.2.97.142", 20338)
		toAddr := payload.NewNetAddress("127.0.0.1", 20338)

		messageVer := payload.NewMsgVersion(protocol.ProtocolVersion, fromAddr, toAddr)

		if err != nil {
			t.Fatal(err)
		}

		if err := wire.WriteMessage(conn, protocol.DevNet, messageVer); err != nil {
			t.Fatal(err)
			return
		}

		readmsg, err := wire.ReadMessage(conn, protocol.DevNet)
		if err != nil {
			t.Fatal(err)
		}
		version, ok := readmsg.(*payload.MsgVersion)
		if !ok {
			t.Fatal(err)
		}

		assert.NotEqual(t, nil, version)

		messageVrck := payload.NewMsgVerAck()
		if err != nil {
			t.Fatal(err)
		}

		assert.NotEqual(t, nil, messageVrck)

		if err := wire.WriteMessage(conn, protocol.DevNet, messageVrck); err != nil {
			t.Fatal(err)
		}

		readmsg, err = wire.ReadMessage(conn, protocol.DevNet)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotEqual(t, nil, readmsg)

		verk, ok := readmsg.(*payload.MsgVerAck)
		if !ok {
			t.Fatal(err)
		}
		assert.NotEqual(t, nil, verk)

		return
	}

}

func TestResponseHandler(t *testing.T) {
	_, conn := net.Pipe()

	inbound := true

	rspHndlr := createResponseHandler()

	p := peermgr.NewPeer(conn, inbound, rspHndlr)

	// test inbound
	assert.Equal(t, inbound, p.Inbound())

	// handshake not done, should be false
	assert.Equal(t, false, p.IsVerackReceived())

	assert.WithinDuration(t, time.Now(), p.CreatedAt(), 1*time.Second)
}

func TestHandshakeCancelled(t *testing.T) {
	// These are the conditions which should invalidate the handshake.
	// Make sure peer is disconnected.
}

func TestPeerDisconnect(t *testing.T) {
	// Make sure everything is shutdown
	// Make sure timer is shutdown in stall detector too. Should maybe put this part of test into stall detector.

	_, conn := net.Pipe()
	inbound := true
	rspHndlr := createResponseHandler()
	p := peermgr.NewPeer(conn, inbound, rspHndlr)

	p.Disconnect()
	verack := payload.NewMsgVerAck()
	err := p.Write(verack)

	assert.NotEqual(t, err, nil)

	// Check if Stall detector is still running
	_, ok := <-p.Detector.Quitch
	assert.Equal(t, ok, false)

}

func TestNotifyDisconnect(t *testing.T) {

	_, conn := net.Pipe()
	inbound := true
	rspHndlr := createResponseHandler()
	p := peermgr.NewPeer(conn, inbound, rspHndlr)

	p.Disconnect()
	p.NotifyDisconnect()
	// TestNotify uses default test timeout as the passing condition
	// Failure condition can be seen when you comment out p.Disconnect()
}

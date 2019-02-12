package peermgr_test

import (
	"io/ioutil"
	"math/rand"
	"net"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

func createConfig() *peermgr.Config {
	return &peermgr.Config{
		Nonce:   120,
		Magic:   protocol.DevNet,
		Handler: createResponseHandler(),
	}
}

func createResponseHandler() peermgr.ResponseHandler {
	OnAddr := func(p *peermgr.Peer, msg *payload.MsgAddr) {}
	OnHeaders := func(p *peermgr.Peer, msg *payload.MsgHeaders) {}
	OnGetHeaders := func(p *peermgr.Peer, msg *payload.MsgGetHeaders) {}
	OnInv := func(p *peermgr.Peer, msg *payload.MsgInv) {}
	OnGetData := func(p *peermgr.Peer, msg *payload.MsgGetData) {}
	OnBlock := func(p *peermgr.Peer, msg *payload.MsgBlock) {}
	OnGetBlocks := func(p *peermgr.Peer, msg *payload.MsgGetBlocks) {}

	return peermgr.ResponseHandler{
		OnHeaders:    OnHeaders,
		OnAddr:       OnAddr,
		OnGetHeaders: OnGetHeaders,
		OnInv:        OnInv,
		OnGetData:    OnGetData,
		OnBlock:      OnBlock,
		OnGetBlocks:  OnGetBlocks,
	}
}

func TestResponseHandler(t *testing.T) {
	_, conn := net.Pipe()
	inbound := true

	p := peermgr.NewPeer(conn, inbound, createConfig())

	// test inbound
	assert.Equal(t, inbound, p.Inbound())
	// handshake not done, should be false
	assert.Equal(t, false, p.IsVerackReceived())
	assert.WithinDuration(t, time.Now(), p.CreatedAt(), 1*time.Second)
}

func TestInboundHandshake(t *testing.T) {
	address := ":20338"

	go func() {
		conn, err := net.DialTimeout("tcp", address, 200*time.Second)
		if err != nil {
			t.Fatal(err)
		}

		p := peermgr.NewPeer(conn, true, createConfig())
		err = p.Run()

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

		version, err := sendAndReadVersion(t, conn, rand.Uint64())
		if err != nil {
			t.Fatal(err)
		}

		assert.NotEqual(t, nil, version)

		msgVerack := payload.NewMsgVerAck()
		if err != nil {
			t.Fatal(err)
		}

		assert.NotEqual(t, nil, msgVerack)

		if err := wire.WriteMessage(conn, protocol.DevNet, msgVerack); err != nil {
			t.Fatal(err)
		}

		readmsg, err := wire.ReadMessage(conn, protocol.DevNet)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotEqual(t, nil, readmsg)

		verack, ok := readmsg.(*payload.MsgVerAck)
		if !ok {
			t.Fatal(err)
		}
		assert.NotEqual(t, nil, verack)
		return
	}
}

func TestOutboundHandshake(t *testing.T) {
	address := ":20338"

	go func() {
		conn, err := net.DialTimeout("tcp", address, 200*time.Second)
		if err != nil {
			t.Fatal(err)
		}

		p := peermgr.NewPeer(conn, false, createConfig())
		err = p.Run()
		if err != nil {
			t.Fatal(err)
		}
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

		version, err := sendAndReadVersion(t, conn, rand.Uint64())
		if err != nil {
			t.Fatal(err)
		}

		assert.NotEqual(t, nil, version)

		messageVer := payload.NewMsgVerAck()
		if err := wire.WriteMessage(conn, protocol.DevNet, messageVer); err != nil {
			t.Fatal(err)
			return
		}

		readmsg, err := wire.ReadMessage(conn, protocol.DevNet)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotEqual(t, nil, readmsg)

		verack, ok := readmsg.(*payload.MsgVerAck)
		if !ok {
			t.Fatal(err)
		}
		assert.NotEqual(t, nil, verack)
		return
	}
}

// TestHandshakeCancelled tests the response message after sending a 'version'
func TestHandshakeCancelled(t *testing.T) {
	address := ":20338"

	go func() {
		conn, err := net.DialTimeout("tcp", address, 200*time.Second)
		if err != nil {
			t.Fatal(err)
		}

		p := peermgr.NewPeer(conn, true, createConfig())
		err = p.Run()
		if err != nil {
			assert.NotEqual(t, nil, err)
		}
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

		if err != nil {
			t.Fatal(err)
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

		// A 'version' came in, let's send a wrong msg response
		messageVer := payload.NewMsgVerAck()
		if err := wire.WriteMessage(conn, protocol.DevNet, messageVer); err != nil {
			t.Fatal(err)
			return
		}
		return
	}
}

// TestHandshakeWrongVersion tests a peer returning a wrong version.
func TestHandshakeWrongVersion(t *testing.T) {
	// Make sure peer is disconnected.
	address := ":20338"

	go func() {
		conn, err := net.DialTimeout("tcp", address, 200*time.Second)
		if err != nil {
			t.Fatal(err)
		}

		p := peermgr.NewPeer(conn, true, createConfig())
		err = p.Run()
		if err != nil {
			assert.NotEqual(t, nil, err)
		}
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

		if err != nil {
			t.Fatal(err)
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

		fromAddr := payload.NewNetAddress("", 20338)
		toAddr := payload.NewNetAddress("", 20338)

		pver := &protocol.Version{
			Major: 1,
			Minor: 0,
			Patch: 0,
		}

		messageVer := payload.NewMsgVersion(pver, fromAddr, toAddr, protocol.FullNode, rand.Uint64())

		if err := wire.WriteMessage(conn, protocol.DevNet, messageVer); err != nil {
			t.Fatal(err)
			return
		}

		readmsg, err = wire.ReadMessage(conn, protocol.DevNet)
		if err != nil {
			t.Fatal(err)
		}
		_, ok = readmsg.(*payload.MsgReject)
		if !ok {
			t.Fatal(err)
		}
		return
	}
}

// TestHandshakeNoVerack tests a peer returning no verack as last message.
func TestHandshakeNoVerack(t *testing.T) {
	address := ":20338"

	go func() {
		conn, err := net.DialTimeout("tcp", address, 200*time.Second)
		if err != nil {
			t.Fatal(err)
		}

		p := peermgr.NewPeer(conn, false, createConfig())
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		err = p.Run()
		if err != nil {
			assert.NotEqual(t, nil, err)
		}

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

		version, err := sendAndReadVersion(t, conn, rand.Uint64())
		if err != nil {
			t.Fatal(err)
			return
		}

		assert.NotEqual(t, nil, version)

		_, err = wire.ReadMessage(conn, protocol.DevNet)
		if err != nil {
			assert.NotEqual(t, nil, err)
		}
		return
	}
}

// TestHandshakeSelfConnect tests a peer receiving a msg from itself.
func TestHandshakeSelfConnect(t *testing.T) {
	address := ":20338"
	cfg := createConfig()
	go func() {
		conn, err := net.DialTimeout("tcp", address, 200*time.Second)
		if err != nil {
			t.Fatal(err)
		}

		p := peermgr.NewPeer(conn, false, cfg)
		err = p.Run()
		if err != nil {
			assert.NotEqual(t, nil, err)
		}
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

		if _, err := sendAndReadVersion(t, conn, cfg.Nonce); err != nil {
			assert.NotEqual(t, nil, err)
		}

		return
	}
}

func sendAndReadVersion(t *testing.T, conn net.Conn, nonce uint64) (*payload.MsgVersion, error) {
	fromAddr := payload.NewNetAddress("", 20338)
	toAddr := payload.NewNetAddress("", 20338)

	msgVersion := payload.NewMsgVersion(protocol.NodeVer, fromAddr, toAddr, protocol.FullNode, nonce)
	if err := wire.WriteMessage(conn, protocol.DevNet, msgVersion); err != nil {
		return nil, err
	}

	_, err := wire.ReadMessage(conn, protocol.DevNet)
	if err != nil {
		return nil, err
	}

	return msgVersion, nil
}

// TestPeerDisconnect
// Make sure everything is shutdown
// Make sure timer is shutdown in stall detector too. Should maybe put this part of test into stall detector.
func TestPeerDisconnect(t *testing.T) {
	_, conn := net.Pipe()
	inbound := true

	p := peermgr.NewPeer(conn, inbound, createConfig())

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

	p := peermgr.NewPeer(conn, inbound, createConfig())

	p.Disconnect()
	p.NotifyDisconnect()
	// TestNotify uses default test timeout as the passing condition
	// Failure condition can be seen when you comment out p.Disconnect()
}

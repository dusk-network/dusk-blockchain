package notifications

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type mockWebsocketConn struct {
	msgBuf map[string]bool
}

func (c *mockWebsocketConn) WriteMessage(messageType int, data []byte) error {

	// Mimic connection consuming the json msg
	var p BlockMsg
	if err := json.Unmarshal(data, &p); err == nil {
		c.msgBuf[p.Hash] = true
	}

	return nil
}
func (c *mockWebsocketConn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	return nil
}
func (c *mockWebsocketConn) RemoteAddr() net.Addr {
	return &mockAddr{addr: "0.0.0.0"}
}
func (c *mockWebsocketConn) SetWriteDeadline(t time.Time) error {
	return nil
}
func (c *mockWebsocketConn) Close() error {
	return nil
}

func (c *mockWebsocketConn) NextReader() (messageType int, r io.Reader, err error) {
	return 0, r, errors.New("no impl")
}

type mockAddr struct {
	addr string
}

func (m *mockAddr) Network() string {
	return "mocked"
}
func (m *mockAddr) String() string {
	return m.addr
}

func TestPoolBasicScenario(t *testing.T) {

	eb := eventbus.New()
	pool := NewPool(eb, 10, 100)
	ctxActiveConn := make([]*mockWebsocketConn, 100)

	// Simulate HTTP server pushing new websocket connections
	for i := 0; i < len(ctxActiveConn); i++ {
		conn := &mockWebsocketConn{}
		conn.msgBuf = make(map[string]bool)
		pool.ConnectionsChan <- conn

		ctxActiveConn[i] = conn
	}

	time.Sleep(1 * time.Second)

	// Simulate eventBus publishing sample acceptedBlocks
	ctxSentMsgs := make([]string, 0)
	for height := 0; height < 20; height++ {

		blk := helper.RandomBlock(t, uint64(height), 3)
		hash, _ := blk.CalculateHash()
		blk.Header.Hash = hash

		msg := message.New(topics.AcceptedBlock, *blk)
		eb.Publish(topics.AcceptedBlock, msg)
		ctxSentMsgs = append(ctxSentMsgs, hex.EncodeToString(blk.Header.Hash))
	}

	time.Sleep(1 * time.Second)

	// Ensure all clients have received all published blocks
	for i := 0; i < len(ctxSentMsgs); i++ {

		sentMsg := ctxSentMsgs[i]
		// Ensure sentMsg is received by each conn
		for cInd := 0; cInd < len(ctxActiveConn); cInd++ {
			conn := ctxActiveConn[cInd]
			_, ok := conn.msgBuf[sentMsg]
			if !ok {
				t.Fatalf("Not all messages have been received by all clients")
			}
		}
	}

	if len(ctxSentMsgs) == 0 || len(ctxActiveConn) == 0 {
		t.Fatal("invalid test context")
	}
}

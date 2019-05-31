package monitor

import (
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/eventmon/logger"
)

var _shutdown = struct{}{}

const unixSoc = "/tmp/dusk-socket"

func TestClient(t *testing.T) {
	defer func() {
		os.Remove(unixSoc)
	}()

	endChan := make(chan struct{})
	msgChan := spinSrv(endChan, unixSoc)
	// waiting for the server to be up and running
	<-msgChan

	conn, err := net.Dial("unix", unixSoc)
	assert.NoError(t, err)

	testMsg := mockAggro()
	logProc := logger.New(conn, nil)
	logProc.PublishRoundEvent(testMsg)

	result := <-msgChan

	assert.Equal(t, "monitor", result["process"])
	assert.Equal(t, float64(3), result["step"])
	assert.Equal(t, float64(23), result["round"])
	shutdown(conn)
	<-endChan
}

func shutdown(conn net.Conn) {
	if err := conn.Close(); err != nil {
		panic(err)
	}
}

func mockAggro() []byte {
	k1, _ := user.NewRandKeys()
	k2, _ := user.NewRandKeys()

	blockHash, _ := crypto.RandEntropy(32)
	aggro := agreement.MockAgreement(blockHash, uint64(23), uint8(3), []user.Keys{k1, k2})
	return aggro.Bytes()
}

func spinSrv(endChan chan<- struct{}, addr string) <-chan map[string]interface{} {
	resChan := make(chan map[string]interface{})

	server := newServer(addr)
	go func() {
		var err error
		server.srv, err = net.Listen("unix", server.addr)
		if err != nil {
			panic(err)
		}
		resChan <- nil
		conn, err := server.srv.Accept()
		// notifying that the server can accept connections
		if err != nil {
			panic(err)
		}
		if conn == nil {
			panic("Connection is nil")
		}

		var msg map[string]interface{}
		// we create a decoder that reads directly from the socket
		d := json.NewDecoder(conn)

		if err := d.Decode(&msg); err != nil {
			panic(err)
		}

		resChan <- msg
		time.AfterFunc(time.Second, func() {
			endChan <- _shutdown
			conn.Close()
		})
	}()
	return resChan
}

// server holds the structure of our TCP
// implementation.
type server struct {
	addr string
	srv  net.Listener
}

func newServer(addr string) *server {
	return &server{addr: addr}
}

// Close shuts down the TCP Server
func (t *server) Close() (err error) {
	return t.srv.Close()
}

package monitor

// import (
// 	"bytes"
// 	"fmt"
// 	"math/big"
// 	"net"
// 	"testing"
// 	"time"

// 	"github.com/bwesterb/go-ristretto"
// 	"github.com/stretchr/testify/assert"
// 	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
// 	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
// 	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
// 	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
// 	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
// )

// const _shutdown = "__shutdown!!"
// const _started = "__started!!"

// func TestClient(t *testing.T) {
// 	endChan := make(chan struct{})
// 	msgChan := spinSrv(endChan, ":1234")
// 	initMsg := <-msgChan
// 	assert.Equal(t, _started, initMsg)
// 	inputChan, quitChan, conn := newClient(":1234")
// 	inputChan <- "pippo"

// 	received := <-msgChan
// 	assert.Equal(t, "pippo\n", received)
// 	shutdown(conn)
// 	<-endChan
// 	quitChan <- _empty
// }

// func TestInfo(t *testing.T) {
// 	endChan := make(chan struct{})
// 	round := uint64(2)
// 	hash, _ := crypto.RandEntropy(32)
// 	resChan := spinSrv(endChan, ":1233")
// 	initMsg := <-resChan
// 	assert.Equal(t, _started, initMsg)
// 	bus := wire.NewEventBus()
// 	d := ristretto.Scalar{}
// 	d.SetBigInt(big.NewInt(100))
// 	broker := LaunchMonitor(bus, ":1233", d)
// 	consensus.UpdateRound(bus, round)

// 	<-time.After(1 * time.Second)

// 	keys := make([]*user.Keys, 2)
// 	for i := 0; i < 2; i++ {
// 		keys[i], _ = user.NewRandKeys()
// 	}
// 	agreement.PublishMock(bus, hash, round, 2, keys)
// 	agreementMsg := <-resChan
// 	consensus.UpdateRound(bus, round+1)

// 	assert.Equal(t, "status:agreement\n", agreementMsg)

// 	// info := <-resChan
// 	<-resChan

// 	shutdown(broker.conn)
// 	<-endChan
// }

// func shutdown(conn net.Conn) {
// 	fmt.Fprint(conn, _shutdown)
// 	if err := conn.Close(); err != nil {
// 		panic(err)
// 	}
// }

// func spinSrv(endChan chan<- struct{}, addr string) <-chan string {
// 	resChan := make(chan string)

// 	server := newServer(addr)
// 	go func() {
// 		var err error
// 		server.srv, err = net.Listen("tcp", server.addr)
// 		if err != nil {
// 			panic(err)
// 		}
// 		resChan <- _started
// 		conn, err := server.srv.Accept()
// 		// notifying that the server can accept connections
// 		for {
// 			if err != nil {
// 				panic(err)
// 			}
// 			if conn == nil {
// 				panic("Connection is nil")
// 			}

// 			defer conn.Close()
// 			out := make([]byte, 1024)
// 			if _, err := conn.Read(out); err == nil {
// 				n := bytes.IndexByte(out, 0)
// 				msg := string(out[:n])
// 				if msg == _shutdown {
// 					endChan <- _empty
// 					return
// 				}
// 				resChan <- msg

// 			} else {
// 				panic("could not read from connection")
// 			}
// 		}
// 	}()
// 	return resChan
// }

// // server holds the structure of our TCP
// // implementation.
// type server struct {
// 	addr string
// 	srv  net.Listener
// }

// func newServer(addr string) *server {
// 	return &server{addr: addr}
// }

// // Close shuts down the TCP Server
// func (t *server) Close() (err error) {
// 	return t.srv.Close()
// }

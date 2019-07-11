package monitor_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/eventmon/logger"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/eventmon/monitor"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

const unixSoc = "unix:///tmp/dusk-socket"

func TestLogger(t *testing.T) {
	msgChan, addr := initTest()
	eb := wire.NewEventBus()
	conn, err := net.Dial("unix", addr)
	assert.NoError(t, err)

	testMsg := mockAggro(23)
	logProc := logger.New(eb, conn, nil)
	logProc.PublishRoundEvent(testMsg)

	result := <-msgChan

	assert.Equal(t, "monitor", result["process"])
	assert.Equal(t, float64(3), result["step"])
	assert.Equal(t, float64(23), result["round"])

	_ = conn.Close()
}

func TestSupervisor(t *testing.T) {
	msgChan, _ := initTest()
	eb := wire.NewEventBus()
	supervisor, err := monitor.Launch(eb, unixSoc)
	assert.NoError(t, err)

	testMsg := mockAggroMsg(23, topics.Agreement)
	// testing that we can receive messages
	eb.Stream(string(topics.Gossip), bytes.NewBuffer(testMsg))
	result := <-msgChan

	assert.Equal(t, "monitor", result["process"])
	assert.Equal(t, float64(3), result["step"])
	assert.Equal(t, float64(23), result["round"])
	_ = supervisor.Stop()
}

func TestSupervisorReconnect(t *testing.T) {
	msgChan, addr := initTest()
	eb := wire.NewEventBus()
	supervisor, err := monitor.Launch(eb, unixSoc)
	assert.NoError(t, err)

	testMsg := mockAggroMsg(23, topics.Agreement)
	// testing that we can receive messages
	eb.Stream(string(topics.Gossip), bytes.NewBuffer(testMsg))
	<-msgChan

	assert.NoError(t, supervisor.Stop())

	testMsg = mockAggroMsg(24, topics.Agreement)
	eb.Stream(string(topics.Gossip), bytes.NewBuffer(testMsg))
	select {
	case <-msgChan:
		assert.FailNow(t, "Expected the supervised LogProcessor to be closed")
	case <-time.After(1 * time.Second):
		// all fine
	}

	// restarting the server as the Stop has likely killed it
	msgChan = startSrv(addr)
	// reconnecting the supervised process
	assert.NoError(t, supervisor.Reconnect())
	// messages streamed when the process is down are lost, so we need to send another message
	eb.Stream(string(topics.Gossip), bytes.NewBuffer(testMsg))
	result := <-msgChan
	assert.Equal(t, "monitor", result["process"])
	assert.Equal(t, float64(3), result["step"])
	assert.Equal(t, float64(24), result["round"])

	_ = supervisor.Stop()
}

func TestResumeRight(t *testing.T) {
	msgChan, _ := initTest()
	eb := wire.NewEventBus()
	supervisor, err := monitor.Launch(eb, unixSoc)
	assert.NoError(t, err)

	testMsg := mockAggroMsg(23, topics.Agreement)
	eb.Stream(string(topics.Gossip), bytes.NewBuffer(testMsg))
	round1 := <-msgChan
	if _, ok := round1["blockTime"]; ok {
		assert.FailNow(t, "First round should not really have a block time. Instead found %d", round1["blockTime"])
	}

	time.Sleep(time.Second)
	testMsg = mockAggroMsg(24, topics.Agreement)
	eb.Stream(string(topics.Gossip), bytes.NewBuffer(testMsg))
	round2 := <-msgChan

	assert.InDelta(t, float64(1000), round2["blockTime"], float64(100))

	_ = supervisor.Stop()
}

func TestNotifyErrors(t *testing.T) {
	endChan := make(chan struct{})
	msgChan, _ := initTest()
	eb := wire.NewEventBus()
	supervisor, err := monitor.Launch(eb, unixSoc)
	assert.NoError(t, err)

	log.AddHook(supervisor)
	log.Errorln("pippo")

	// wrapped in a go routing to check that there are no race conditions
	go func() {
		msg := <-msgChan
		assert.Equal(t, "error", msg["level"])
		assert.Equal(t, "pippo", msg["msg"])
		endChan <- struct{}{}
	}()

	testMsg := mockAggroMsg(23, topics.Agreement)
	eb.Stream(string(topics.Gossip), bytes.NewBuffer(testMsg))
	result := <-msgChan
	assert.Equal(t, "monitor", result["process"])
	<-endChan
}

func initTest() (<-chan map[string]interface{}, string) {
	addr := unixSocPath()
	_ = os.Remove(addr)
	msgChan := startSrv(addr)
	return msgChan, addr
}

func startSrv(addr string) <-chan map[string]interface{} {
	msgChan := spinSrv(addr)
	// waiting for the server to be up and running
	<-msgChan
	return msgChan
}

func unixSocPath() string {
	uri, err := url.Parse(unixSoc)
	if err != nil {
		panic(err)
	}
	return uri.Path
}

func mockAggro(round uint64) []byte {
	k1, _ := user.NewRandKeys()
	k2, _ := user.NewRandKeys()
	ks := []user.Keys{k1, k2}
	blockHash, _ := crypto.RandEntropy(32)

	aggro := agreement.MockAgreement(blockHash, round, uint8(3), ks)
	return aggro.Bytes()
}

func mockAggroMsg(round uint64, tpc topics.Topic) []byte {
	b := mockAggro(round)
	buf, _ := wire.AddTopic(bytes.NewBuffer(b), tpc)
	return buf.Bytes()
}

func spinSrv(addr string) <-chan map[string]interface{} {
	resChan := make(chan map[string]interface{})

	go func() {
		var conn net.Conn
		srv, err := net.Listen("unix", addr)
		if err != nil {
			panic(err)
		}
		resChan <- nil

		conn, err = srv.Accept()
		// notifying that the server can accept connections
		if err != nil {
			panic(err)
		}

		if conn == nil {
			panic("Connection is nil")
		}

		// we create a decoder that reads directly from the socket
		d := json.NewDecoder(conn)
		for {
			var msg map[string]interface{}
			if err := d.Decode(&msg); err == io.EOF {
				break
			} else if err != nil {
				panic(err)
			}

			resChan <- msg
		}
		_ = conn.Close()
		srv.Close()
	}()

	return resChan
}

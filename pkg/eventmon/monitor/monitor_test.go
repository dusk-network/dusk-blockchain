package monitor_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/logger"
	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/monitor"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/stretchr/testify/assert"
)

const unixSoc = "unix:///tmp/dusk-socket"

func TestLogger(t *testing.T) {
	msgChan, addr, wg := initTest()
	eb := eventbus.New()
	conn, err := net.Dial("unix", addr)
	assert.NoError(t, err)

	testBlk := helper.RandomBlock(t, 23, 4)
	logProc := logger.New(eb, conn, nil)
	logProc.PublishBlockEvent(testBlk)

	result := <-msgChan

	assert.Equal(t, "monitor", result["process"])
	assert.Equal(t, float64(23), result["round"])

	_ = logProc.Close()
	wg.Wait()
}

func TestSupervisor(t *testing.T) {
	msgChan, _, wg := initTest()
	eb := eventbus.New()
	supervisor, err := monitor.Launch(eb, unixSoc)
	assert.NoError(t, err)

	testBuf := mockBlockBuf(t, 23)
	// testing that we can receive messages
	eb.Publish(topics.AcceptedBlock, testBuf)
	result := <-msgChan

	assert.Equal(t, "monitor", result["process"])
	assert.Equal(t, float64(23), result["round"])
	_ = supervisor.Stop()
	wg.Wait()
}

func TestSupervisorReconnect(t *testing.T) {
	msgChan, addr, wg := initTest()
	eb := eventbus.New()
	supervisor, err := monitor.Launch(eb, unixSoc)
	assert.NoError(t, err)

	testBuf := mockBlockBuf(t, 23)
	// testing that we can receive messages
	eb.Publish(topics.AcceptedBlock, testBuf)
	<-msgChan

	assert.NoError(t, supervisor.Stop())
	wg.Wait()

	testBuf = mockBlockBuf(t, 24)
	eb.Publish(topics.AcceptedBlock, testBuf)
	select {
	case <-msgChan:
		assert.FailNow(t, "Expected the supervised LogProcessor to be closed")
	case <-time.After(1 * time.Second):
		// all fine
	}

	// restarting the server as the Stop has likely killed it
	msgChan, wg = startSrv(addr)
	// reconnecting the supervised process
	assert.NoError(t, supervisor.Reconnect())
	// messages streamed when the process is down are lost, so we need to send another message
	eb.Publish(topics.AcceptedBlock, testBuf)
	result := <-msgChan
	assert.Equal(t, "monitor", result["process"])
	assert.Equal(t, float64(24), result["round"])

	_ = supervisor.Stop()
	wg.Wait()
}

func TestResumeRight(t *testing.T) {
	msgChan, _, wg := initTest()
	eb := eventbus.New()
	supervisor, err := monitor.Launch(eb, unixSoc)
	assert.NoError(t, err)

	testBuf := mockBlockBuf(t, 23)
	eb.Publish(topics.AcceptedBlock, testBuf)
	round1 := <-msgChan
	if _, ok := round1["blockTime"]; ok {
		assert.FailNow(t, "First round should not really have a block time. Instead found %d", round1["blockTime"])
	}

	time.Sleep(3 * time.Second)

	// Publish next block
	testBuf = mockBlockBuf(t, 24)
	eb.Publish(topics.AcceptedBlock, testBuf)
	var round2 map[string]interface{}
	for {
		// If we get a message, discard it if it is not a block event message
		round2 = <-msgChan
		if round2["blockTime"] != nil {
			break
		}
	}

	assert.GreaterOrEqual(t, round2["blockTime"], float64(3), float64(1))

	_ = supervisor.Stop()
	wg.Wait()
}

func TestNotifyErrors(t *testing.T) {
	endChan := make(chan struct{})
	msgChan, _, wg := initTest()
	eb := eventbus.New()
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

	testBuf := mockBlockBuf(t, 23)
	eb.Publish(topics.AcceptedBlock, testBuf)
	result := <-msgChan
	assert.Equal(t, "monitor", result["process"])
	<-endChan
	_ = supervisor.Stop()
	wg.Wait()
}

// Test that we properly drop log entries instead of hanging when the buffer is full
func TestHook(t *testing.T) {
	// We do not need the msgChan, as we are simulating a frozen monitoring process
	// Neither do we need the waitgroup, since waiting for this server to stop will
	// block indefinitely.
	_, _, _ = initTest()
	eb := eventbus.New()
	supervisor, err := monitor.Launch(eb, unixSoc)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	defer log.SetOutput(os.Stderr)
	log.AddHook(supervisor)

	// Log 1000 events, and notify when done
	doneChan := make(chan struct{}, 1)
	go func() {
		for i := 0; i < 1000; i++ {
			log.Errorln("pippo")
		}
		doneChan <- struct{}{}
	}()

	// Fail fast if we get stuck
	select {
	case <-doneChan:
	case <-time.After(2 * time.Second):
		t.Fatal("logging events took too long")
	}

	_ = supervisor.Stop()
}

// Test that setting the deadline on Send does not interfere
// with other logging operations (for instance the accepted block
// event)
func TestDeadline(t *testing.T) {
	msgChan, _, wg := initTest()
	eb := eventbus.New()
	supervisor, err := monitor.Launch(eb, unixSoc)
	assert.NoError(t, err)

	log.AddHook(supervisor)

	// Send an error entry, to trigger Send
	log.Errorln("pippo")

	msg := <-msgChan
	assert.Equal(t, "error", msg["level"])
	assert.Equal(t, "pippo", msg["msg"])

	// The write deadline is 3 seconds, so let's wait for that to expire
	time.Sleep(3 * time.Second)

	testBuf := mockBlockBuf(t, 23)
	eb.Publish(topics.AcceptedBlock, testBuf)

	// Should get the accepted block message on the msgchan
	for {
		msg = <-msgChan
		// We should discard any other messages
		if msg["code"] == "round" {
			// Success
			break
		}
	}

	_ = supervisor.Stop()
	wg.Wait()
}

func mockBlockBuf(t *testing.T, height uint64) *bytes.Buffer {
	blk := helper.RandomBlock(t, height, 4)
	buf := new(bytes.Buffer)
	if err := marshalling.MarshalBlock(buf, blk); err != nil {
		panic(err)
	}

	return buf
}

func initTest() (<-chan map[string]interface{}, string, *sync.WaitGroup) {
	addr := unixSocPath()
	_ = os.Remove(addr)
	msgChan, wg := startSrv(addr)
	return msgChan, addr, wg
}

func startSrv(addr string) (<-chan map[string]interface{}, *sync.WaitGroup) {
	msgChan, wg := spinSrv(addr)
	// waiting for the server to be up and running
	<-msgChan
	return msgChan, wg
}

func unixSocPath() string {
	uri, err := url.Parse(unixSoc)
	if err != nil {
		panic(err)
	}
	return uri.Path
}

func spinSrv(addr string) (<-chan map[string]interface{}, *sync.WaitGroup) {
	resChan := make(chan map[string]interface{}, 5)
	wg := &sync.WaitGroup{}

	go func() {
		wg.Add(1)
		var conn net.Conn
		srv, err := net.Listen("unix", addr)
		if err != nil {
			panic(err)
		}

		// notifying that the server can accept connections
		resChan <- nil
		conn, err = srv.Accept()
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
		wg.Done()
	}()

	return resChan, wg
}

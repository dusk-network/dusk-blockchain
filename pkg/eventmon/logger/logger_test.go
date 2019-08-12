package logger_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"sync"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/logger"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
)

const unixSoc = "unix:///tmp/dusk-socket"

var withTimeTest = []struct {
	fields   log.Fields
	msg      string
	expected []string
}{
	{nil, "pippo", []string{"time", "process"}},
	{log.Fields{"topolino": "pluto"}, "pippo", []string{"time", "process", "topolino"}},
}

func TestWithTime(t *testing.T) {
	for _, tt := range withTimeTest {
		eb := wire.NewEventBus()
		// setup
		b := &BufCloser{new(bytes.Buffer)}
		logBase, data := setup(eb, nil, b)

		// tested function
		logBase.WithTime(tt.fields).Info(tt.msg)
		assert.NoError(t, json.Unmarshal(b.Bytes(), &data))

		// checking msg
		assert.Equal(t, tt.msg, data["msg"])
		// checking that the fields are there
		for _, exp := range tt.expected {
			_, ok := data[exp]
			assert.True(t, ok)
		}
	}
}

func TestSendDeadlock(t *testing.T) {
	bus := wire.NewEventBus()

	addr, err := url.Parse(unixSoc)
	if err != nil {
		t.Fatal(err)
	}

	// start a server
	wg, c := newServer(addr)
	// wait for the server to be ready to accept conns
	wg.Wait()

	conn, err := net.Dial(addr.Scheme, addr.Path)
	if err != nil {
		t.Fatal(err)
	}

	// Make sure the buffer fills up after one message
	if unixConn, ok := conn.(*net.UnixConn); ok {
		unixConn.SetWriteBuffer(1)
	}

	// setup
	logBase, _ := setup(bus, nil, conn)

	// log enough to fill up the buffer
	for i := 0; i < 4; i++ {
		for _, tt := range withTimeTest {
			entry := logBase.WithTime(tt.fields)
			err = logBase.Send(entry)
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	// clean up hanging goroutine
	c <- struct{}{}
}

func setup(eb wire.EventBroker, formatter log.Formatter, wc io.WriteCloser) (*logger.LogProcessor, map[string]interface{}) {
	var data map[string]interface{}
	logBase := logger.New(eb, wc, formatter)
	return logBase, data
}

type BufCloser struct {
	*bytes.Buffer
}

func (b *BufCloser) Close() error {
	return nil
}

func newServer(addr *url.URL) (*sync.WaitGroup, chan struct{}) {
	_ = os.Remove(addr.Path)
	l, err := net.Listen(addr.Scheme, addr.Path)
	if err != nil {
		panic(err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	c := make(chan struct{})

	go func() {
		wg.Done()
		_, err := l.Accept()
		if err != nil {
			panic(err)
		}
		<-c
	}()

	return wg, c
}

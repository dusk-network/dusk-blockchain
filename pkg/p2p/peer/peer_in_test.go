package peer

import (
	"bytes"
	"net"
	"os"
	"testing"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/stretchr/testify/require"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	// suppressing logging due to expected errors
	logrus.SetLevel(logrus.FatalLevel)
}

// Test that the 'ping' message is sent correctly, and that a 'pong' message will result.
func TestPingLoop(t *testing.T) {
	// suppressing expected error message about method not registered
	bus := eventbus.New()
	client, srv := net.Pipe()

	//setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := cfg.LoadFromFile(cwd + "/../../../dusk.toml")
	require.Nil(t, err)

	// override keepAliveTime
	r.Timeout.TimeoutKeepAliveTime = 1

	cfg.Mock(&r)

	responseChan := make(chan bytes.Buffer, 10)
	writer := NewWriter(client, processing.NewGossip(protocol.TestNet), bus)
	go writer.Serve(responseChan, make(chan struct{}, 1))

	// Set up the other end of the exchange
	responseChan2 := make(chan bytes.Buffer, 10)
	writer2 := NewWriter(srv, processing.NewGossip(protocol.TestNet), bus)
	go writer2.Serve(responseChan2, make(chan struct{}, 1))

	// Set up reader factory
	processor := NewMessageProcessor(bus)
	processor.Register(topics.Ping, responding.ProcessPing)
	factory := NewReaderFactory(processor)

	reader, err := factory.SpawnReader(client, processing.NewGossip(protocol.TestNet), dupemap.NewDupeMap(0), responseChan, make(chan struct{}, 1))
	if err != nil {
		t.Fatal(err)
	}
	go reader.ReadLoop()

	reader2, err := factory.SpawnReader(srv, processing.NewGossip(protocol.TestNet), dupemap.NewDupeMap(0), responseChan2, make(chan struct{}, 1))
	if err != nil {
		t.Fatal(err)
	}
	go reader2.ReadLoop()

	// We should eventually get a pong message out of responseChan2
	for {
		buf := <-responseChan2
		topic, err := topics.Extract(&buf)
		if err != nil {
			t.Fatal(err)
		}

		if topics.Pong.String() == topic.String() {
			break
		}
	}
}

// TestIncompleteChecksum ensures peer reader does not panic on
// incomplete checksum buffer
func TestIncompleteChecksum(t *testing.T) {
	bus := eventbus.New()

	// Set up reader factory
	processor := NewMessageProcessor(bus)
	factory := NewReaderFactory(processor)

	peer, _, w, _ := testReader(t, factory)
	defer func() {
		_ = peer.Close()
	}()

	// Construct an invalid frame
	frame := new(bytes.Buffer)
	// Add length bytes
	if err := encoding.WriteUint64LE(frame, 7); err != nil {
		t.Error(err)
	}

	// Add correct magic value
	mBuf := protocol.TestNet.ToBuffer()
	if _, err := frame.Write(mBuf.Bytes()); err != nil {
		t.Error(err)
	}

	// Checksum with 3 bytes only
	if _, err := frame.Write([]byte{0, 0, 0}); err != nil {
		t.Error(err)
	}

	_, err := w.Write(frame.Bytes())
	if err != nil {
		t.Error(err)
	}

	time.Sleep(100 * time.Millisecond)
}

// TestZeroLength ensures peer reader does not panic on 0 length field
func TestZeroLength(t *testing.T) {
	bus := eventbus.New()

	// Set up reader factory
	processor := NewMessageProcessor(bus)
	factory := NewReaderFactory(processor)

	peer, _, w, _ := testReader(t, factory)
	defer func() {
		_ = peer.Close()
	}()

	// Construct an invalid frame
	frame := new(bytes.Buffer)
	// Add 0 length (any value smaller than 4)
	if err := encoding.WriteUint64LE(frame, 0); err != nil {
		t.Error(err)
	}

	// Add correct magic value
	mBuf := protocol.TestNet.ToBuffer()
	if _, err := frame.Write(mBuf.Bytes()); err != nil {
		t.Error(err)
	}

	_, err := w.Write(frame.Bytes())
	if err != nil {
		t.Error(err)
	}

	time.Sleep(100 * time.Millisecond)
}

// TestOverflowLength
// Ensure peer reader does not panic overflow length value
func TestOverflowLength(t *testing.T) {
	bus := eventbus.New()

	// Set up reader factory
	processor := NewMessageProcessor(bus)
	factory := NewReaderFactory(processor)

	peer, _, w, _ := testReader(t, factory)
	defer func() {
		_ = peer.Close()
	}()

	// Construct an invalid frame with large length field
	frame := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(frame, 999999999); err != nil {
		t.Error(err)
	}

	_, err := w.Write(frame.Bytes())
	if err != nil {
		t.Error(err)
	}

	time.Sleep(100 * time.Millisecond)
}

// TestInvalidPayload
// Ensure peer reader does not panic on sending invalid payload to any topic
func TestInvalidPayload(t *testing.T) {
	bus := eventbus.New()

	// Set up reader factory
	processor := NewMessageProcessor(bus)
	factory := NewReaderFactory(processor)

	peer, _, w, _ := testReader(t, factory)
	defer func() {
		_ = peer.Close()
	}()

	// Send for all possible topic values with a malformed
	// payload
	var topic byte
	for topic = 0; topic <= 50; topic++ {
		buf := &bytes.Buffer{}
		buf.WriteByte(topic)
		buf.Write([]byte{0, 1, 2})

		cs := checksum.Generate(buf.Bytes())
		_ = processing.WriteFrame(buf, protocol.TestNet, cs)

		_, err := w.Write(buf.Bytes())
		if err != nil {
			t.Error(err)
		}

		time.Sleep(20 * time.Millisecond)
	}
}

//nolint:unparam
func testReader(t *testing.T, f *ReaderFactory) (*Reader, net.Conn, net.Conn, chan<- bytes.Buffer) {
	d := dupemap.NewDupeMap(0)
	r, w := net.Pipe()

	respChan := make(chan bytes.Buffer, 10)
	g := processing.NewGossip(protocol.TestNet)
	peer, _ := f.SpawnReader(r, g, d, respChan, make(chan struct{}, 1))

	// Run the non-recover readLoop to watch for panics
	go assert.NotPanics(t, peer.readLoop)

	time.Sleep(200 * time.Millisecond)

	return peer, r, w, respChan
}

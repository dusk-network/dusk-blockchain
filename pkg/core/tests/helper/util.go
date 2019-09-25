package helper

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
	"github.com/stretchr/testify/assert"
)

// TxsToBuffer converts a slice of transactions to a bytes.Buffer.
func TxsToBuffer(t *testing.T, txs []transactions.Transaction) *bytes.Buffer {
	buf := new(bytes.Buffer)

	for _, tx := range txs {
		err := transactions.Marshal(buf, tx)
		if err != nil {
			assert.Nil(t, err)
		}
	}

	return bytes.NewBuffer(buf.Bytes())
}

// RandomSlice returns a random slice of size `size`
func RandomSlice(t *testing.T, size uint32) []byte {
	randSlice := make([]byte, size)
	_, err := rand.Read(randSlice)
	assert.Nil(t, err)
	return randSlice
}

// SimpleStreamer is a test helper which can capture information that gets gossiped
// by the node. It can read from the gossip stream, and stores the topics that it has
// seen.
type SimpleStreamer struct {
	lock       sync.RWMutex
	seenTopics []topics.Topic
	*bufio.Reader
	*bufio.Writer
}

// NewSimpleStreamer returns an initialized SimpleStreamer.
func NewSimpleStreamer() *SimpleStreamer {
	r, w := io.Pipe()
	return &SimpleStreamer{
		seenTopics: make([]topics.Topic, 0),
		Reader:     bufio.NewReader(r),
		Writer:     bufio.NewWriter(w),
	}
}

func (ms *SimpleStreamer) Write(p []byte) (n int, err error) {
	n, err = ms.Writer.Write(p)
	if err != nil {
		return n, err
	}

	return n, ms.Writer.Flush()
}

func (ms *SimpleStreamer) Read() ([]byte, error) {
	bs, err := processing.ReadFrame(ms.Reader)
	if err != nil {
		return nil, err
	}

	decoded := bytes.NewBuffer(bs)

	// read and discard the magic
	magicBuf := make([]byte, 4)
	if _, err := decoded.Read(magicBuf); err != nil {
		return nil, err
	}

	// check the topic
	topicBuffer := make([]byte, 15)
	if _, err := decoded.Read(topicBuffer); err != nil {
		return nil, err
	}

	var cmd [15]byte
	copy(cmd[:], topicBuffer)
	ms.lock.Lock()
	ms.seenTopics = append(ms.seenTopics, topics.ByteArrayToTopic(cmd))
	ms.lock.Unlock()

	return decoded.Bytes(), nil
}

// SeenTopics returns a slice of all the topics the SimpleStreamer has found in its
// stream so far.
func (ms *SimpleStreamer) SeenTopics() []topics.Topic {
	ms.lock.RLock()
	defer ms.lock.RUnlock()
	return ms.seenTopics
}

// Close implements io.WriteCloser.
func (ms *SimpleStreamer) Close() error {
	return nil
}

// CreateGossipStreamer sets up and event bus, subscribes a SimpleStreamer to the
// gossip topic, and sets the right preprocessors up for the gossip topic.
func CreateGossipStreamer() (*eventbus.EventBus, *SimpleStreamer) {
	eb := eventbus.New()
	eb.RegisterPreprocessor(string(topics.Gossip), processing.NewGossip(protocol.TestNet))
	// subscribe to gossip topic
	streamer := NewSimpleStreamer()
	eb.SubscribeStream(string(topics.Gossip), streamer)

	return eb, streamer
}

func StartPeerReader(conn net.Conn, bus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, counter *chainsync.Counter, responseChan chan<- *bytes.Buffer) (*peer.Reader, error) {
	dupeMap := dupemap.NewDupeMap(5)
	exitChan := make(chan struct{}, 1)
	return peer.NewReader(conn, protocol.TestNet, dupeMap, bus, rpcBus, counter, responseChan, exitChan)
}

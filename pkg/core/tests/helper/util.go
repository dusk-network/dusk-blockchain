package helper

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// TxsToReader converts a slice of transactions to an io.Reader
func TxsToReader(t *testing.T, txs []transactions.Transaction) io.Reader {
	buf := new(bytes.Buffer)

	for _, tx := range txs {
		err := tx.Encode(buf)
		if err != nil {
			assert.Nil(t, err)
		}
	}

	return bytes.NewReader(buf.Bytes())
}

// RandomSlice returns a random slice of size `size`
func RandomSlice(t *testing.T, size uint32) []byte {
	randSlice := make([]byte, size)
	_, err := rand.Read(randSlice)
	assert.Nil(t, err)
	return randSlice
}

type SimpleStreamer struct {
	sync.RWMutex
	seenTopics []topics.Topic
	*bufio.Reader
	*bufio.Writer
}

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
	buf, err := ms.Reader.ReadBytes(0x00)
	if err != nil {
		return nil, err
	}

	decoded := peer.Decode(buf)

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
	ms.Lock()
	ms.seenTopics = append(ms.seenTopics, topics.ByteArrayToTopic(cmd))
	ms.Unlock()

	return decoded.Bytes(), nil
}

func (ms *SimpleStreamer) SeenTopics() []topics.Topic {
	ms.RLock()
	defer ms.RUnlock()
	return ms.seenTopics
}

func (ms *SimpleStreamer) Close() error {
	return nil
}

func CreateGossipStreamer() (*wire.EventBus, *SimpleStreamer) {
	eb := wire.NewEventBus()
	eb.RegisterPreprocessor(string(topics.Gossip), peer.NewGossip(protocol.TestNet))
	// subscribe to gossip topic
	streamer := NewSimpleStreamer()
	eb.SubscribeStream(string(topics.Gossip), streamer)

	return eb, streamer
}

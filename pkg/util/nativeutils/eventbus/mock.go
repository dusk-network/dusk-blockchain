package eventbus

import (
	"bufio"
	"bytes"
	"io"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

var _ wire.EventCollector = (*Collector)(nil)

// Collector is a very stupid implementation of the wire.EventCollector interface
// in case no function would be supplied, it would use a channel to publish the collected packets
type Collector struct {
	f func(bytes.Buffer) error
}

// NewSimpleCollector is a simple wrapper around a callback that redirects collected buffers into a channel
func NewSimpleCollector(rChan chan bytes.Buffer, f func(bytes.Buffer) error) *Collector {
	if f == nil {
		f = func(b bytes.Buffer) error {
			rChan <- b
			return nil
		}
	}
	return &Collector{f}
}

// Collect redirects a buffer copy to a channel
func (m *Collector) Collect(b bytes.Buffer) error {
	return m.f(b)
}

var _ Preprocessor = (*Adder)(nil)

// Adder is a very simple Preprocessor for test purposes
type Adder struct {
	token string
}

// NewAdder creates a new Adder
func NewAdder(tkn string) *Adder {
	return &Adder{tkn}
}

// Process a buffer by appending a string to it
func (a *Adder) Process(buf *bytes.Buffer) error {
	buf.WriteString(a.token)
	return nil
}

// CreateGossipStreamer sets up and event bus, subscribes a SimpleStreamer to the
// gossip topic, and sets the right preprocessors up for the gossip topic.
func CreateGossipStreamer() (*EventBus, *SimpleStreamer) {
	eb := New()
	eb.Register(string(topics.Gossip), processing.NewGossip(protocol.TestNet))
	// subscribe to gossip topic
	streamer := NewSimpleStreamer()
	streamListener := NewStreamListener(streamer)
	eb.Subscribe(string(topics.Gossip), streamListener)

	return eb, streamer
}

// SimpleStreamer is a WriteCloser
var _ io.WriteCloser = (*SimpleStreamer)(nil)

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

	return decoded.Bytes(), nil
}

type GossipStreamer struct {
	*SimpleStreamer
}

func NewGossipStreamer() *GossipStreamer {
	return &GossipStreamer{
		SimpleStreamer: NewSimpleStreamer(),
	}
}

func (ms *GossipStreamer) Read() ([]byte, error) {
	b, err := ms.SimpleStreamer.Read()
	if err != nil {
		return nil, err
	}

	decoded := bytes.NewBuffer(b)

	// check the topic
	topicBuffer := make([]byte, 15)
	if _, err = decoded.Read(topicBuffer); err != nil {
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
func (ms *GossipStreamer) SeenTopics() []topics.Topic {
	ms.lock.RLock()
	defer ms.lock.RUnlock()
	return ms.seenTopics
}

// Close implements io.WriteCloser.
func (ms *SimpleStreamer) Close() error {
	return nil
}

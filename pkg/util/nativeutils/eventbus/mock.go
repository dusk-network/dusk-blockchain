package eventbus

import (
	"bufio"
	"bytes"
	"io"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
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
func CreateGossipStreamer() (*EventBus, *GossipStreamer) {
	eb := New()
	streamer := NewGossipStreamer(protocol.TestNet)
	streamListener := NewStreamListener(streamer)
	eb.Subscribe(topics.Gossip, streamListener)

	return eb, streamer
}

// CreateFrameStreamer sets up and event bus, subscribes a SimpleStreamer to the
// gossip topic, and sets the right preprocessors up for the gossip topic.
func CreateFrameStreamer(topic topics.Topic) (*EventBus, io.WriteCloser) {
	eb := New()
	streamer := NewSimpleStreamer(protocol.TestNet)
	streamListener := NewStreamListener(streamer)
	eb.Subscribe(topic, streamListener)

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
	gossip *processing.Gossip
}

// NewSimpleStreamer returns an initialized SimpleStreamer.
func NewSimpleStreamer(magic protocol.Magic) *SimpleStreamer {
	r, w := io.Pipe()
	return &SimpleStreamer{
		seenTopics: make([]topics.Topic, 0),
		Reader:     bufio.NewReader(r),
		Writer:     bufio.NewWriter(w),
		gossip:     processing.NewGossip(magic),
	}
}

// Write receives the packets from the ringbuffer. It performs a Gossip.Process
// (since there is no longer a preprocessor) before writing to the internal pipe
func (ms *SimpleStreamer) Write(p []byte) (n int, err error) {
	b := bytes.NewBuffer(p)
	if err := ms.gossip.Process(b); err != nil {
		return 0, err
	}

	n, err = ms.Writer.Write(b.Bytes())
	if err != nil {
		return n, err
	}

	return n, ms.Writer.Flush()
}

func (ms *SimpleStreamer) Read() ([]byte, error) {
	length, err := ms.gossip.UnpackLength(ms.Reader)
	if err != nil {
		return nil, err
	}

	packet := make([]byte, length)
	if read, err := io.ReadFull(ms.Reader, packet); err != nil {
		return nil, err
	} else if uint64(read) != length {
		return nil, io.EOF
	}
	return packet, nil
}

type GossipStreamer struct {
	*SimpleStreamer
}

func NewGossipStreamer(magic protocol.Magic) *GossipStreamer {
	return &GossipStreamer{
		SimpleStreamer: NewSimpleStreamer(magic),
	}
}

func (ms *GossipStreamer) Read() ([]byte, error) {
	b, err := ms.SimpleStreamer.Read()
	if err != nil {
		return nil, err
	}

	decoded := bytes.NewBuffer(b)

	// remove checksum
	m, _, err := checksum.Extract(decoded.Bytes())
	if err != nil {
		return nil, err
	}

	// check the topic
	decoded = bytes.NewBuffer(m)
	topic, err := topics.Extract(decoded)
	if err != nil {
		return nil, err
	}

	ms.lock.Lock()
	ms.seenTopics = append(ms.seenTopics, topic)
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

package eventbus

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// Collector is a very stupid implementation of the wire.EventCollector interface
// in case no function would be supplied, it would use a channel to publish the collected packets
type Collector struct {
	f func(message.Message) error
}

// NewSimpleCollector is a simple wrapper around a callback that redirects collected buffers into a channel
func NewSimpleCollector(rChan chan message.Message, f func(message.Message) error) *Collector {
	if f == nil {
		f = func(m message.Message) error {
			rChan <- m
			return nil
		}
	}
	return &Collector{f}
}

// Collect redirects a buffer copy to a channel
func (m *Collector) Collect(b message.Message) error {
	return m.f(b)
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
var _ io.WriteCloser = (*StupidStreamer)(nil)

// StupidStreamer is a streamer meant for using when testing internal
// forwarding of binary packets through the ring buffer. It does *not* add
// magic, frames or other wraps on the forwarded packet
type StupidStreamer struct {
	*bufio.Reader
	*bufio.Writer
}

// NewStupidStreamer returns an initialized SimpleStreamer.
func NewStupidStreamer() *StupidStreamer {
	r, w := io.Pipe()
	return &StupidStreamer{
		Reader: bufio.NewReader(r),
		Writer: bufio.NewWriter(w),
	}
}

// Write receives the packets from the ringbuffer and writes it on the internal
// pipe immediatelyh
func (sss *StupidStreamer) Write(p []byte) (n int, err error) {
	lg := make([]byte, 4)
	binary.LittleEndian.PutUint16(lg, uint16(len(p)))
	b := bytes.NewBuffer(lg)
	b.Write(p)

	n, err = sss.Writer.Write(b.Bytes())
	if err != nil {
		return n, err
	}

	return n, sss.Writer.Flush()
}

func (sss *StupidStreamer) Read() ([]byte, error) {
	length := make([]byte, 4)
	if _, err := io.ReadFull(sss.Reader, length); err != nil {
		return nil, err
	}

	l := binary.LittleEndian.Uint16(length)
	payload := make([]byte, l)
	if _, err := io.ReadFull(sss.Reader, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

// Close implements io.WriteCloser.
func (sss *StupidStreamer) Close() error {
	return nil
}

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

// Write receives the packets from the ringbuffer and writes it on the internal
// pipe immediatelyh
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
	fmt.Println(topic)

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

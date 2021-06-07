// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package eventbus

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
)

// Collector is a very stupid implementation of the wire.EventCollector interface
// in case no function would be supplied, it would use a channel to publish the collected packets.
type Collector struct {
	f func(message.Message) error
}

// NewSimpleCollector is a simple wrapper around a callback that redirects collected buffers into a channel.
func NewSimpleCollector(rChan chan message.Message, f func(message.Message) error) *Collector {
	if f == nil {
		f = func(m message.Message) error {
			rChan <- m
			return nil
		}
	}

	return &Collector{f}
}

// Collect redirects a buffer copy to a channel.
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
func CreateFrameStreamer(topic topics.Topic) (*EventBus, ring.Writer) {
	eb := New()
	streamer := NewSimpleStreamer(protocol.TestNet)
	streamListener := NewStreamListener(streamer)
	eb.Subscribe(topic, streamListener)
	return eb, streamer
}

// SimpleStreamer is a WriteCloser.
var (
	_ ring.Writer    = (*SimpleStreamer)(nil)
	_ io.WriteCloser = (*StupidStreamer)(nil)
)

// StupidStreamer is a streamer meant for using when testing internal
// forwarding of binary packets through the ring buffer. It does *not* add
// magic, frames or other wraps on the forwarded packet.
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
// pipe immediately.
func (sss *StupidStreamer) Write(p []byte) (n int, err error) {
	lg := make([]byte, 4)
	binary.LittleEndian.PutUint32(lg, uint32(len(p)))
	b := bytes.NewBuffer(lg)
	_, _ = b.Write(p)

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

	l := binary.LittleEndian.Uint32(length)

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
	//nolint:structcheck
	lock       sync.RWMutex
	seenTopics []topics.Topic
	*bufio.Reader
	*bufio.Writer
	gossip *protocol.Gossip
}

// NewSimpleStreamer returns an initialized SimpleStreamer.
func NewSimpleStreamer(magic protocol.Magic) *SimpleStreamer {
	r, w := io.Pipe()

	return &SimpleStreamer{
		seenTopics: make([]topics.Topic, 0),
		Reader:     bufio.NewReader(r),
		Writer:     bufio.NewWriter(w),
		gossip:     protocol.NewGossip(magic),
	}
}

// Write receives the packets from the ringbuffer and writes it on the internal
// pipe immediately.
func (ms *SimpleStreamer) Write(p []byte, h []byte, t topics.Topic, priority ring.MsgPriority) (n int, err error) {
	b := bytes.NewBuffer(p)
	if e := ms.gossip.Process(b); e != nil {
		return 0, e
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

// GossipStreamer is a SimpleStreamer which removes the checksum and the topic
// when reading. It is supposed to be used when testing data that needs to be
// streamed over the network.
type GossipStreamer struct {
	*SimpleStreamer
}

// NewGossipStreamer creates a new GossipStreamer instance.
func NewGossipStreamer(magic protocol.Magic) *GossipStreamer {
	return &GossipStreamer{
		SimpleStreamer: NewSimpleStreamer(magic),
	}
}

// Read the stream.
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

// RouterStreamer reroutes a gossiped message to a list of EventBus instances.
type RouterStreamer struct {
	// list of peers eventBus instances to route msg to
	lock  sync.Mutex
	peers []*EventBus
}

// NewRouterStreamer instantiate RouterStreamer with empty list.
func NewRouterStreamer() *RouterStreamer {
	return &RouterStreamer{peers: make([]*EventBus, 0)}
}

// Add adds a dest eventBus.
func (r *RouterStreamer) Add(p *EventBus) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.peers = append(r.peers, p)
}

// Write implements io.WriteCloser.
func (r *RouterStreamer) Write(p []byte, h []byte) (n int, err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	// Re-route message to all peers
	for i := 0; i < len(r.peers); i++ {
		b := bytes.NewBuffer(p)

		msg, err := message.Unmarshal(b)
		if err != nil {
			return 0, err
		}

		r.peers[i].Publish(msg.Category(), msg)
	}

	return len(p), nil
}

func (r *RouterStreamer) Read() ([]byte, error) {
	return nil, nil
}

// Close implements io.WriteCloser.
func (r *RouterStreamer) Close() error {
	return nil
}

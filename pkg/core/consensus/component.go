package consensus

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// Priority indicates a rough order among components subscribed to the same topic
type Priority uint8

const (
	// HighPriority indicates precedence
	HighPriority Priority = iota
	// LowPriority indicates that the component should be the last to get updates
	LowPriority
)

// Signer encapsulate the credentials to sign or authenticate outgoing events
type Signer interface {
	// Sign a payload. The first is parameter is a block hash
	Sign(header.Header) ([]byte, error)

	// Gossip concatenates all information before gossiping it to the
	// rest of the network.
	// It accepts a topic, a blockhash, a payload and the ID of the requesting
	// component
	Gossip(message.Message, uint32) error

	// Compose is used to inject authentication data to a component specific
	// packet. It is supposed to be used whenever a component needs to create a
	// **new** Packet for internal propagation
	Compose(PacketFactory) InternalPacket

	// SendInternally is used for internal forwarding
	SendInternally(topics.Topic, message.Message, uint32) error
}

// EventPlayer is the interface used by Components to signal their intention to
// get, pause or resume events for a given Step
type EventPlayer interface {
	// Forward signals the Coordinator that a component wishes to further the step
	// of the consensus. An ID needs to be supplied in order for the Coordinator to
	// decide if this request is valid.
	Forward(uint32) uint8
	// Pause signals the Coordinator to temporarily pause Event forwarding for
	// a Listener specified through its ID.
	Pause(uint32)
	// Play resumes the Event forwarding for a Listener with the given ID.
	Play(uint32)
}

// ComponentFactory holds the data to create a Component (i.e. Signer, EventPublisher, RPCBus). Its responsibility is to recreate it on demand
type ComponentFactory interface {
	// Instantiate a new Component without initializing it
	Instantiate() Component
}

// Component is an ephemeral instance that lives solely for a round
type Component interface {
	// Initialize a Component with data relevant to the current Round
	Initialize(EventPlayer, Signer, RoundUpdate) []TopicListener
	// Finalize allows a Component to perform cleanup operations before begin garbage collected
	Finalize()
	// ID allows the Coordinator to differentiate between components and
	// establish relevance or problems
	ID() uint32

	// Name
	Name() string
}

// Listener subscribes to the Coordinator and forwards consensus events to the components
type Listener interface {
	// NotifyPayload forwards consensus events to the component
	NotifyPayload(InternalPacket) error
	// ID is used to later unsubscribe from the Coordinator. This is useful for components active throughout
	// multiple steps
	ID() uint32
	// Priority indicates the Priority of a Listener
	Priority() Priority
	Paused() bool
	Pause()
	Resume()
}

// SimpleListener implements Listener and uses a callback for notifying events
type SimpleListener struct {
	callback func(InternalPacket) error
	id       uint32
	priority Priority
	lock     sync.RWMutex
	paused   bool
}

// NewSimpleListener creates a SimpleListener
func NewSimpleListener(callback func(InternalPacket) error, priority Priority, paused bool) Listener {
	// #654
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	id := binary.LittleEndian.Uint32(buf)

	return &SimpleListener{callback, id, priority, sync.RWMutex{}, paused}
}

// NotifyPayload triggers the callback specified during instantiation
func (s *SimpleListener) NotifyPayload(ev InternalPacket) error {
	return s.callback(ev)
}

// ID returns the id to allow Component to unsubscribe
func (s *SimpleListener) ID() uint32 {
	return s.id
}

// Priority as indicated by the Listener interface
func (s *SimpleListener) Priority() Priority {
	return s.priority
}

// Paused returns whether this Listener is Paused
func (s *SimpleListener) Paused() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.paused
}

// Pause the SimpleListener
func (s *SimpleListener) Pause() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.paused = true
}

// Resume the SimpleListener
func (s *SimpleListener) Resume() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.paused = false
}

// FilteringListener is a Listener that performs filtering before triggering the callback specified by the component
// Normally it is used to filter out events sent by Provisioners not being part of a committee or invalid messages.
// Filtering is applied to the `header.Header`
type FilteringListener struct {
	*SimpleListener
	filter func(header.Header) bool
}

// NewFilteringListener creates a FilteringListener
func NewFilteringListener(callback func(InternalPacket) error, filter func(header.Header) bool, priority Priority, paused bool) Listener {
	// #654
	nBig, err := rand.Int(rand.Reader, big.NewInt(32))
	if err != nil {
		panic(err)
	}
	n := nBig.Int64()
	id := uint32(n)
	return &FilteringListener{&SimpleListener{callback, id, priority, sync.RWMutex{}, paused}, filter}
}

// NotifyPayload uses the filtering function to let only relevant events through
func (cb *FilteringListener) NotifyPayload(ev InternalPacket) error {
	hdr := ev.State()
	if cb.filter(hdr) {
		return fmt.Errorf("event has been filtered and won't be forwarded to the component - round: %d / step: %d", hdr.Round, hdr.Step)
	}
	return cb.SimpleListener.NotifyPayload(ev)
}

// TopicListener is Listener carrying a Topic
type TopicListener struct {
	Listener
	Topic topics.Topic
}

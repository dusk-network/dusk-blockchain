package consensus

import (
	"bytes"
	"fmt"
	"math/rand"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
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
	Sign([]byte, []byte) ([]byte, error)
	// SendAuthenticated performs a ED25519 signature on a message before forwarding
	// It accepts a topic, a blockhash, a payload and the ID of the requesting
	// component
	SendAuthenticated(topics.Topic, []byte, *bytes.Buffer, uint32) error
	// SendWithHeader is used for internal forwarding. It exposes the same
	// parameters as SendAuthenticated but does not perform a ED25519 signature
	// on the Event (and neither forwards it to the Gossip topic
	SendWithHeader(topics.Topic, []byte, *bytes.Buffer, uint32) error
}

// EventPlayer is the interface used by Components to signal their intention to
// get, pause or resume events for a given Step
type EventPlayer interface {
	// Play signals the Coordinator that a newly initialized Component is ready
	// to accept new `Event`. Components need to indicate their ID to let the
	// Coordinator identify duplicates or obsolete components.
	// It returns the current `Step`
	Play(uint32) uint8
	// Pause signals the Coordinator to temporarily pause Event forwarding for
	// a Listener specified through its ID
	Pause(uint32)
	// Resume the Event forwarding for a Listener
	Resume(uint32)
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
}

// Listener subscribes to the Coordinator and forwards consensus events to the components
type Listener interface {
	// NotifyPayload forwards consensus events to the component
	NotifyPayload(Event) error
	// ID is used to later unsubscribe from the Coordinator. This is useful for components active throughout
	// multiple steps
	ID() uint32
	// Priority indicates the Priority of a Listener
	Priority() Priority
}

// SimpleListener implements Listener and uses a callback for notifying events
type SimpleListener struct {
	callback func(Event) error
	id       uint32
	priority Priority
}

// NewSimpleListener creates a SimpleListener
func NewSimpleListener(callback func(Event) error, priority Priority) Listener {
	id := rand.Uint32()
	return &SimpleListener{callback, id, priority}
}

// NotifyPayload triggers the callback specified during instantiation
func (s *SimpleListener) NotifyPayload(ev Event) error {
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

// FilteringListener is a Listener that performs filtering before triggering the callback specified by the component
// Normally it is used to filter out events sent by Provisioners not being part of a committee or invalid messages.
// Filtering is applied to the `header.Header`
type FilteringListener struct {
	*SimpleListener
	filter func(header.Header) bool
}

// NewFilteringListener creates a FilteringListener
func NewFilteringListener(callback func(Event) error, filter func(header.Header) bool, priority Priority) Listener {
	id := rand.Uint32()
	return &FilteringListener{&SimpleListener{callback, id, priority}, filter}
}

// NotifyPayload uses the filtering function to let only relevant events through
func (cb *FilteringListener) NotifyPayload(ev Event) error {
	if cb.filter(ev.Header) {
		return fmt.Errorf("event has been filtered and won't be forwarded to the component - round: %d / step: %d", ev.Header.Round, ev.Header.Step)
	}
	return cb.SimpleListener.NotifyPayload(ev)
}

// TopicListener is Listener carrying a Topic
type TopicListener struct {
	Listener
	Preprocessors []eventbus.Preprocessor
	Topic         topics.Topic
	Paused        bool
}

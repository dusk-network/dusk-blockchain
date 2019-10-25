package consensus

import (
	"bytes"
	"errors"
	"math/rand"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// Stepper requests a Step update
type Stepper interface {
	// RequestStepUpdate is used by components to signal the termination of some processing.
	// It is used to avoid manipulating the Step directly in the Component since this is shared
	// across all components
	RequestStepUpdate()
}

// Signer encapsulate the credentials to sign or authenticate outgoing events
type Signer interface {
	Sign([]byte, []byte) ([]byte, error)
	SendAuthenticated(topics.Topic, []byte, *bytes.Buffer) error
	SendWithHeader(topics.Topic, []byte, *bytes.Buffer) error
}

// Subscriber to the Coordinator. It mimics the eventbus.Subscriber but uses a different kind of Listener
// more suitable to the consensus Components
type Subscriber interface {
	Subscribe(topics.Topic, Listener)
	Unsubscribe(uint32)
}

// ComponentFactory holds the data to create a Component (i.e. Signer, EventPublisher, RPCBus). Its responsibility is to recreate it on demand
type ComponentFactory interface {
	// Instantiate a new Component without initializing it
	Instantiate() Component
}

// Component is an ephemeral instance that lives solely for a round
type Component interface {
	// Initialize a Component with data relevant to the current Round
	Initialize(Stepper, Signer, Subscriber, RoundUpdate) []TopicListener
	// Finalize allows a Component to perform cleanup operations before begin garbage collected
	Finalize()
}

// Listener subscribes to the Coordinator and forwards consensus events to the components
type Listener interface {
	// NotifyPayload forwards consensus events to the component
	NotifyPayload(Event) error
	// ID is used to later unsubscribe from the Coordinator. This is useful for components active throughout
	// multiple steps
	ID() uint32
}

// SimpleListener implements Listener and uses a callback for notifying events
type SimpleListener struct {
	callback func(Event) error
	id       uint32
}

// NewSimpleListener creates a SimpleListener
func NewSimpleListener(callback func(Event) error) Listener {
	id := rand.Uint32()
	return &SimpleListener{callback, id}
}

// NotifyPayload triggers the callback specified during instantiation
func (s *SimpleListener) NotifyPayload(ev Event) error {
	return s.callback(ev)
}

// ID returns the id to allow Component to unsubscribe
func (s *SimpleListener) ID() uint32 {
	return s.id
}

// FilteringListener is a Listener that performs filtering before triggering the callback specified by the component
// Normally it is used to filter out events sent by Provisioners not being part of a committee or invalid messages.
// Filtering is applied to the `header.Header`
type FilteringListener struct {
	*SimpleListener
	filter func(header.Header) bool
}

// NewFilteringListener creates a FilteringListener
func NewFilteringListener(callback func(Event) error, filter func(header.Header) bool) Listener {
	id := rand.Uint32()
	return &FilteringListener{&SimpleListener{callback, id}, filter}
}

// NotifyPayload uses the filtering function to let only relevant events through
func (cb *FilteringListener) NotifyPayload(ev Event) error {
	if cb.filter(ev.Header) {
		return errors.New("event has been filtered and won't be forwarded to the component")
	}
	return cb.SimpleListener.NotifyPayload(ev)
}

// TopicListener is Listener carrying a Topic
type TopicListener struct {
	Listener
	Preprocessors []eventbus.Preprocessor
	Topic         topics.Topic
}

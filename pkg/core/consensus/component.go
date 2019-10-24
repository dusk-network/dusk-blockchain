package consensus

import (
	"bytes"
	"errors"
	"math/rand"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

type Stepper interface {
	RequestStepUpdate()
}
type Signer interface {
	Sign([]byte, []byte) ([]byte, error)
	SendAuthenticated(topics.Topic, []byte, *bytes.Buffer) error
}

type Subscriber interface {
	Subscribe(topics.Topic, Listener)
	Unsubscribe(uint32)
}

// ComponentFactory holds the data to create a Component (i.e. Signer, EventPublisher, RPCBus). Its responsibility is to recreate it on demand
type ComponentFactory interface {
	Instantiate() Component
}

// Component is an ephemeral instance that lives solely for a round
type Component interface {
	Initialize(Stepper, Signer, RoundUpdate) []TopicListener
	Finalize()
}

type Listener interface {
	NotifyPayload(Event) error
	ID() uint32
}

type SimpleListener struct {
	callback func(Event) error
	id       uint32
}

func NewSimpleListener(callback func(Event) error) (Listener, uint32) {
	id := rand.Uint32()
	return &SimpleListener{callback, id}, id
}

func (s *SimpleListener) NotifyPayload(ev Event) error {
	return s.callback(ev)
}

func (s *SimpleListener) ID() uint32 {
	return s.id
}

type FilteringListener struct {
	*SimpleListener
	filter func(header.Header) bool
}

func NewFilteringListener(callback func(Event) error, filter func(header.Header) bool) (Listener, uint32) {
	id := rand.Uint32()
	return &FilteringListener{&SimpleListener{callback, id}, filter}, id
}

func (cb *FilteringListener) NotifyPayload(ev Event) error {
	if cb.filter(ev.Header) {
		return errors.New("event has been filtered and won't be forwarded to the component")
	}
	return cb.SimpleListener.NotifyPayload(ev)
}

type TopicListener struct {
	Listener
	Topic topics.Topic
}

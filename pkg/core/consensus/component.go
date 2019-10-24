package consensus

import (
	"bytes"
	"errors"

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

// ComponentFactory holds the data to create a Component (i.e. Signer, EventPublisher, RPCBus). Its responsibility is to recreate it on demand
type ComponentFactory interface {
	Instantiate() Component
}

// Component is an ephemeral instance that lives solely for a round
type Component interface {
	Initialize(Stepper, Signer, RoundUpdate) []Subscriber
	Finalize()
	SetStep(uint8)
}

type Listener interface {
	NotifyPayload(Event) error
}

type SimpleListener struct {
	callback func(Event) error
}

func NewSimpleListener(callback func(Event) error) Listener {
	return &SimpleListener{callback}
}

func (s *SimpleListener) NotifyPayload(ev Event) error {
	return s.callback(ev)
}

type FilteringListener struct {
	*SimpleListener
	filter func(header.Header) bool
}

func NewFilteringListener(callback func(Event) error, filter func(header.Header) bool) Listener {
	return &FilteringListener{&SimpleListener{callback}, filter}
}

func (cb *FilteringListener) NotifyPayload(ev Event) error {
	if cb.filter(ev.Header) {
		return errors.New("event has been filtered and won't be forwarded to the component")
	}
	return cb.SimpleListener.NotifyPayload(ev)
}

type Subscriber struct {
	Listener
	Topic topics.Topic
}

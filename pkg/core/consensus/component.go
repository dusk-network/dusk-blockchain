package consensus

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

type Store interface {
	RequestStepUpdate()
	RequestSignature([]byte) ([]byte, error)
	WriteHeader([]byte, *bytes.Buffer) error
}

// ComponentFactory holds the data to create a Component (i.e. Signer, EventPublisher, RPCBus). Its responsibility is to recreate it on demand
type ComponentFactory interface {
	Instantiate() Component
}

// Component is an ephemeral instance that lives solely for a round
type Component interface {
	Initialize(Store, RoundUpdate) []Subscriber
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
		return nil
	}
	return cb.SimpleListener.NotifyPayload(ev)
}

type Signer struct {
	keys      user.Keys
	consensus *Consensus
}

type Subscriber struct {
	Listener
	Topic topics.Topic
}

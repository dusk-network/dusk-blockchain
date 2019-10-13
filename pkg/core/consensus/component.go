package consensus

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-crypto/bls"
)

type Store interface {
	RequestStepUpdate()
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

func (s *Signer) BLSSign(payload []byte) ([]byte, error) {
	round, step := s.consensus.Round(), s.consensus.Step()
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, round); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buf, step); err != nil {
		return nil, err
	}

	if _, err := buf.Write(payload); err != nil {
		return nil, err
	}

	signedHash, err := bls.Sign(s.keys.BLSSecretKey, s.keys.BLSPubKey, payload)
	if err != nil {
		return nil, err
	}

	return signedHash.Compress(), nil
}

type Subscriber struct {
	Listener
	topic topics.Topic
}

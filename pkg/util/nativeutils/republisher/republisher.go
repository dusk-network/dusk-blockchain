package republisher

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

var (
	DuplicatePayloadError = errors.New("duplicate payload")
	InvalidError          = errors.New("invalid payload")
	EncodingError         = errors.New("encoding failed")
)

type (
	Validator func(bytes.Buffer) error

	// Republisher handles the repropagation of messages propagated with a
	// specified topic
	Republisher struct {
		tpc        topics.Topic
		broker     eventbus.Broker
		id         uint32
		validators []Validator
	}
)

// New creates a Republisher
func New(eb eventbus.Broker, tpc topics.Topic, v ...Validator) *Republisher {
	r := &Republisher{
		broker:     eb,
		tpc:        tpc,
		validators: v,
	}
	r.id = r.Activate()
	return r
}

// Stop a Republisher
func (r *Republisher) Stop() {
	r.broker.Unsubscribe(r.tpc, r.id)
}

// Activate the Republisher by listening to topic through a
// callback listener
func (r *Republisher) Activate() uint32 {
	if r.id != 0 {
		return r.id
	}

	l := eventbus.NewCallbackListener(r.Republish)
	r.id = r.broker.Subscribe(r.tpc, l)
	return r.id
}

// Republish intercepts a topic and repropagates it immediately
// after applying any eventual validation logic
func (r *Republisher) Republish(b bytes.Buffer) error {
	for _, v := range r.validators {
		err := v(b)
		switch err {
		case InvalidError, EncodingError:
			return err
		case DuplicatePayloadError:
			return nil
		}
	}

	if err := topics.Prepend(&b, r.tpc); err != nil {
		return err
	}

	r.broker.Publish(topics.Gossip, &b)
	return nil
}

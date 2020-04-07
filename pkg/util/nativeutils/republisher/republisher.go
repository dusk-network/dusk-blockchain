package republisher

import (
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type (
	// Validator is a function used by the Republisher to establish if the
	// message should be forwarded to the network or otherwise. For instance,
	// the republisher should not propagate transaction being invalid, or
	// messages which Signature check fails, etc
	Validator func(message.Message) error

	// Republisher handles the repropagation of messages propagated with a
	// specified topic
	Republisher struct {
		tpc        topics.Topic
		broker     eventbus.Broker
		id         uint32
		validators []Validator
	}

	repuberr              struct{ err error }
	duplicatePayloadError struct{ *repuberr }
	invalidError          struct{ *repuberr }
	encodingError         struct{ *repuberr }
)

// Error as demanded by the error interface
func (rerr *repuberr) Error() string {
	return rerr.err.Error()
}

// DuplicatePayloadError is the error returned when the republisher detects a
// duplicate
var DuplicatePayloadError = &duplicatePayloadError{&repuberr{errors.New("duplicatePayloadError")}}

// EncodingError is returned when wire un- marshaling fails
var EncodingError = &encodingError{&repuberr{errors.New("encoding failed")}}

// InvalidError is returned when the payload is invalid
var InvalidError = &invalidError{&repuberr{errors.New("invalid payload")}}

// New creates a Republisher for a given topic. Multiple Validator functions
// can be specified for the Republisher to run before forwarding the message
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
// after applying any eventual validation logic.
// Note: the logic for marshaling should be moved after the Gossip
func (r *Republisher) Republish(m message.Message) error {
	for _, v := range r.validators {
		if err := v(m); err != nil {
			switch err {
			case InvalidError, EncodingError:
				return err
			case DuplicatePayloadError:
				return nil
			default:
				return err
			}
		}
	}

	// message.Marshal takes care of prepending the topic, marshaling the
	// header, etc
	buf, err := message.Marshal(m)
	if err != nil {
		return err
	}

	// TODO: interface - setting the payload to a buffer will go away as soon as the Marshaling
	// is performed where it is supposed to (i.e. after the Gossip)
	serialized := message.New(m.Category(), buf)

	// gossip away
	r.broker.Publish(topics.Gossip, serialized)
	return nil
	//r.broker.Publish(topics.Gossip, m)
	//return nil
}

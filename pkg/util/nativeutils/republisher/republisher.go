package republisher

import (
	"bytes"
	"errors"
	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"

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
		safe       bool
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

func newR(eb eventbus.Broker, tpc topics.Topic, v ...Validator) *Republisher {
	return &Republisher{
		broker:     eb,
		tpc:        tpc,
		validators: v,
		safe:       true,
	}
}

// New creates a Republisher for a given topic. Multiple Validator functions
// can be specified for the Republisher to run before forwarding the message.
// The republisher created with New is not thread-safe. Validators need to
// implement their own synchronization. The republisher will reuse the
// CachedBinary whenever possible, without trying to Marshal the message.
// However, if the buffer is empty, it will attempt to Marshal the message,
// which might lead to race-conditions if the Message is accessed by other
// components. In those cases where CachedBinary is not guaranteed to be filled
// (i.e. a message created by this node), or the message payload includes
// references (therefore slices) it is better to create the Republisher through NewSafe
func New(eb eventbus.Broker, tpc topics.Topic, v ...Validator) *Republisher {
	r := newR(eb, tpc, v...)
	r.safe = false
	r.Activate()
	return r
}

// NewSafe creates a safe Republisher which gets a deep copy of the message
func NewSafe(eb eventbus.Broker, tpc topics.Topic, v ...Validator) *Republisher {
	r := newR(eb, tpc, v...)
	r.Activate()
	return r
}

// Stop a Republisher
func (r *Republisher) Stop() {
	r.broker.Unsubscribe(r.tpc, r.id)
}

// Activate the Republisher by listening to topic through a
// callback listener. It saves the ID in the Republisher before returning
func (r *Republisher) Activate() uint32 {
	if r.id != 0 {
		return r.id
	}

	var l eventbus.Listener
	if !r.safe {
		l = eventbus.NewCallbackListener(r.Republish)
	} else {
		l = eventbus.NewSafeCallbackListener(r.Republish)
	}
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

	var marshaled bytes.Buffer
	var err error

	marshaled = m.CachedBinary()
	if marshaled.Len() <= 0 {
		// message.Marshal takes care of prepending the topic, marshaling the
		// header, etc
		marshaled, err = message.Marshal(m)
		if err != nil {
			return err
		}
	}

	// TODO: interface - setting the payload to a buffer will go away as soon as the Marshaling
	// is performed where it is supposed to (i.e. after the Gossip)
	serialized := message.New(m.Category(), marshaled)

	// gossip away
	errList := r.broker.Publish(topics.Gossip, serialized)
	diagnostics.LogPublishErrors("republisher.go, topics.Gossip", errList)

	return nil
	//r.broker.Publish(topics.Gossip, m)
	//return nil
}

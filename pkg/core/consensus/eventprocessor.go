package consensus

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type (
	// Validator is responsible for validating the Ed25519 signature of a message.
	Validator struct{}

	// Republisher is responsible for gossiping a received event buffer.
	Republisher struct {
		publisher eventbus.Publisher
		topic     topics.Topic
	}
)

// NewRepublisher returns a Republisher containing the specified parameters.
func NewRepublisher(publisher eventbus.Publisher, topic topics.Topic) *Republisher {
	return &Republisher{publisher, topic}
}

// Process propagates a received event buffer to other nodes in the network.
func (r *Republisher) Process(eventBuffer *bytes.Buffer) error {
	return r.process(*eventBuffer)
}

func (r *Republisher) process(eventBuffer bytes.Buffer) error {
	if err := topics.Prepend(&eventBuffer, r.topic); err != nil {
		return err
	}

	r.publisher.Publish(topics.Gossip, &eventBuffer)
	return nil
}

// Process a buffer by validating the ED25519 Signature. It uses a io.TeeReader to
// preserve the original message. It returns a copy of the message.
func (v *Validator) Process(buf *bytes.Buffer) error {
	sig := make([]byte, 64)
	if err := encoding.Read512(buf, sig); err != nil {
		return err
	}

	edPubKey := make([]byte, 32)
	if err := encoding.Read256(buf, edPubKey); err != nil {
		return err
	}

	var newBuf bytes.Buffer
	r := io.TeeReader(buf, &newBuf)

	// NOTE: this should really be checked since a gigantic message can crash the machine
	signed, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	if err := msg.VerifyEd25519Signature(edPubKey, signed, sig); err != nil {
		return err
	}

	*buf = newBuf
	return nil
	//return &newBuf, nil
}

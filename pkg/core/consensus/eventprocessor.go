package consensus

import (
	"bytes"
	"io"
	"io/ioutil"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type (
	// Validator is responsible for validating the Ed25519 signature of a message.
	Validator struct{}

	// Republisher is responsible for gossiping a received event buffer.
	Republisher struct {
		publisher wire.EventPublisher
		topic     topics.Topic
	}
)

// NewRepublisher returns a Republisher containing the specified parameters.
func NewRepublisher(publisher wire.EventPublisher, topic topics.Topic) *Republisher {
	return &Republisher{publisher, topic}
}

// Process propagates a received event buffer to other nodes in the network.
func (r *Republisher) Process(eventBuffer *bytes.Buffer) (*bytes.Buffer, error) {
	bounced := eventBuffer
	msg, _ := wire.AddTopic(bounced, r.topic)
	r.publisher.Stream(string(topics.Gossip), msg)
	return eventBuffer, nil
}

// Process a buffer by validating the ED25519 Signature. It uses a io.TeeReader to
// preserve the original message. It returns a copy of the message.
func (v *Validator) Process(buf *bytes.Buffer) (*bytes.Buffer, error) {
	sig := make([]byte, 64)
	if err := encoding.Read512(buf, &sig); err != nil {
		return nil, err
	}

	edPubKey := make([]byte, 32)
	if err := encoding.Read256(buf, &edPubKey); err != nil {
		return nil, err
	}

	var newBuf bytes.Buffer
	r := io.TeeReader(buf, &newBuf)

	// NOTE: this should really be checked since a gigantic message can crash the machine
	signed, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if err := msg.VerifyEd25519Signature(edPubKey, signed, sig); err != nil {
		return nil, err
	}

	return &newBuf, nil
}

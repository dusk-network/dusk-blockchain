package message

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// Message is the core of the message-oriented architecture of the node. It is
// particularly important within the consensus, but in practice any component
// ends up dealing with it. It encapsulates the data exchanged by different
// nodes as well as internal components. The Message transits inside of the
// eventbus and is de- serialized through the Gossip procedure
type Message interface {
	//fmt.Stringer
	Category() topics.Topic
	Payload() interface{}
	Equal(Message) bool
	Id() []byte
}

// Serializable allows to set a payload
type Serializable interface {
	SetPayload(interface{})
}

// SerializableMessage is a Serializable and a Message
type SerializableMessage interface {
	Message
	Serializable
}

// simple is a utility struct that encapsulates the data received by another
// peer and provides protocol-level unmarshaling. It is intended to be
// immutable but also lazy, and therefore it includes the capability to cache the
// Marshaled form
type simple struct {
	// category is normally equivalent to the topic, but can sometimes differ
	// since it actually describes the type of the Payload rather than the
	// subject of the pubsub queue it would be published to (i.e. Gossip)
	category topics.Topic
	// Payload carries the payload of the message, if it can be parsed at
	// protocol level
	payload interface{}
	// cached marshaled form with Category
	marshaled *bytes.Buffer
}

func (m simple) String() string {
	var sb strings.Builder
	_, _ = sb.WriteString("category: ")
	_, _ = sb.WriteString(m.category.String())
	_, _ = sb.WriteString("\n")
	_, _ = sb.WriteString("payload: [\n")
	if m.payload == nil {
		_, _ = sb.WriteString("<payload is nil>")
	} else {
		str, ok := m.payload.(fmt.Stringer)
		if ok {
			_, _ = sb.WriteString(str.String())
		} else {
			_, _ = sb.WriteString("<payload is non-empty but not a Stringer>")
		}
	}
	_, _ = sb.WriteString("\n]\n")
	_, _ = sb.WriteString("\n")
	return sb.String()

}

// Id is the Id the Message
// nolint:golint
func (m simple) Id() []byte {
	if m.marshaled == nil {
		buf, err := Marshal(m)
		if err != nil {
			panic(err)
		}
		m.marshaled = &buf
	}
	return m.marshaled.Bytes()
}

func (m simple) Category() topics.Topic {
	return m.category
}

func (m simple) Payload() interface{} {
	return m.payload
}

func (m *simple) SetPayload(payload interface{}) {
	m.payload = payload
}

func (m simple) Equal(other Message) bool {
	msg, ok := other.(*simple)
	return ok && bytes.Equal(msg.marshaled.Bytes(), msg.marshaled.Bytes())
}

// New creates a new Message
func New(t topics.Topic, payload interface{}) Message {
	return &simple{category: t, payload: payload}
}

//func newMsg(t topics.Topic) *simple {
//	return &simple{category: t}
//}

func (m *simple) initPayloadBuffer(b bytes.Buffer) {
	if m.marshaled == nil {
		m.marshaled = bytes.NewBuffer(b.Bytes())
	}
}

// Unmarshal mutates the buffer by extracting the topic. It create the Message
// by setting the topic and unmarshaling the payload into the proper structure
// It also caches the serialized form within the message
func Unmarshal(b *bytes.Buffer) (Message, error) {
	var err error
	msg := &simple{}
	msg.initPayloadBuffer(*b)

	topic, err := topics.Extract(b)
	if err != nil {
		return nil, err
	}
	msg.category = topic

	switch topic {
	case topics.Tx:
		err = UnmarshalTxMessage(b, msg)
	case topics.Candidate, topics.RoundResults:
		err = UnmarshalCandidateMessage(b, msg)
	case topics.Score:
		err = UnmarshalScoreMessage(b, msg)
	case topics.Reduction:
		err = UnmarshalReductionMessage(b, msg)
	case topics.Agreement:
		err = UnmarshalAgreementMessage(b, msg)
	}

	if err != nil {
		return nil, err
	}

	return *msg, nil
}

// Marshal a Message into a buffer. The buffer *does* include its Category (so
// if this is undesired, the client of this call needs to explicitly call
// topics.Extract on the resulting buffer
// TODO: interface - once the Gossip preprocessing is removed from the
// Coordinator, there won't be a need for marshalBuffer
func Marshal(s Message) (bytes.Buffer, error) {
	var buf bytes.Buffer
	// if it is a simple message, first we check if this message carries a
	// cache of its marshaled form first
	m, ok := s.(simple)

	if !ok { // it is not a cacheable message. We simply marshal without caring for any optimization
		if err := marshal(s, &buf); err != nil {
			return bytes.Buffer{}, err
		}
		return buf, nil
	}

	// it is a simple message
	if m.marshaled != nil {
		// the message has cached its marshaled form
		b := *m.marshaled
		return b, nil
	}

	// this message has never been marshaled before
	if err := marshal(m, &buf); err != nil {
		return bytes.Buffer{}, err
	}

	m.initPayloadBuffer(buf)
	return buf, nil
}

// marshal forces the serialization of a Message regardless of its cache
func marshal(s Message, b *bytes.Buffer) error {
	payload := s.Payload()
	if payload == nil {
		*b = s.Category().ToBuffer()
		return nil
	}

	switch payload := payload.(type) {
	case bytes.Buffer:
		*b = payload

	default:
		if err := marshalMessage(s.Category(), payload, b); err != nil {
			return err
		}
	}

	return topics.Prepend(b, s.Category())
}

// marshalMessage marshals a Message carrying a payload different than a Buffer
// TODO: interface - this should be the actual marshal as soon as the internal
// messages will skip the useless encoding
func marshalMessage(topic topics.Topic, payload interface{}, buf *bytes.Buffer) error {
	var err error
	switch topic {

	case topics.Tx:
		tx := payload.(transactions.Transaction)
		err = MarshalTx(buf, tx)
	case topics.Candidate, topics.RoundResults:
		candidate := payload.(Candidate)
		err = MarshalCandidate(buf, candidate)
	case topics.Score:
		score := payload.(Score)
		err = MarshalScore(buf, score)
	case topics.Reduction:
		reduction := payload.(Reduction)
		err = MarshalReduction(buf, reduction)
	case topics.Agreement:
		agreement := payload.(Agreement)
		err = MarshalAgreement(buf, agreement)
	default:
		return fmt.Errorf("unsupported marshaling of message type: %v", topic.String())
	}

	if err != nil {
		return err
	}

	return nil
}

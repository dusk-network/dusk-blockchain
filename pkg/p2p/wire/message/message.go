// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// EMPTY represents a empty message, used by the StopConsensus call.
const EMPTY = "EMPTY"

// SafeBuffer is a byte.Buffer wrapper that let the Buffer implement
// Safe.
type SafeBuffer struct {
	bytes.Buffer
}

// Copy complies with payload.Safe interface. It returns a deep copy of
// the buffer safe to publish to multiple subscribers.
func (s SafeBuffer) Copy() payload.Safe {
	bin := make([]byte, s.Len())
	copy(bin, s.Bytes())

	b := bytes.NewBuffer(bin)
	return SafeBuffer{*b}
}

// Message is the core of the message-oriented architecture of the node. It is
// particularly important within the consensus, but in practice any component
// ends up dealing with it. It encapsulates the data exchanged by different
// nodes as well as internal components. The Message transits inside of the
// eventbus and is de- serialized through the Gossip procedure.
type Message interface {
	Category() topics.Topic
	Payload() payload.Safe
	Equal(Message) bool
	Id() []byte

	// CachedBinary returns the marshaled form of this Payload as cached during
	// the unmarshaling of incoming messages. In case the message has been
	// created internally and never serialized, this should return an empty buffer.
	CachedBinary() bytes.Buffer

	Header() []byte
}

// Serializable allows to set a payload.
type Serializable interface {
	SetPayload(payload.Safe)
}

// SerializableMessage is a Serializable and a Message.
type SerializableMessage interface {
	Message
	Serializable
}

// simple is a utility struct that encapsulates the data received by another
// peer and provides protocol-level unmarshaling. It is intended to be
// immutable but also lazy, and therefore it includes the capability to cache the
// Marshaled form.
type simple struct {
	// category is normally equivalent to the topic, but can sometimes differ
	// since it actually describes the type of the Payload rather than the
	// subject of the pubsub queue it would be published to (i.e. Gossip).
	category topics.Topic
	// Payload carries the payload of the message, if it can be parsed at
	// protocol level.
	payload payload.Safe
	// cached marshaled form with Category.
	marshaled *bytes.Buffer
	// header used as metadata (e.g kadcast.height).
	header []byte
}

// Clone creates a new Message which carries a copy of the payload.
func Clone(m Message) (Message, error) {
	b := m.CachedBinary()

	if m.Payload() == nil {
		return nil, fmt.Errorf("could not clone message, topic: %s", m.Category())
	}

	return simple{
		category:  m.Category(),
		marshaled: &b,
		payload:   m.Payload().Copy(),
		header:    m.Header(),
	}, nil
}

// CachedBinary complies with the Message method for returning the
// marshaled form of the message if available.
func (m simple) CachedBinary() bytes.Buffer {
	if m.marshaled != nil {
		return *m.marshaled
	}

	return bytes.Buffer{}
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

func (m simple) Header() []byte {
	return m.header
}

// Id is the Id the Message.
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

func (m simple) Payload() payload.Safe {
	return m.payload
}

func (m *simple) SetPayload(payload payload.Safe) {
	m.payload = payload
}

func (m simple) Equal(other Message) bool {
	msg, ok := other.(*simple)
	return ok && bytes.Equal(msg.marshaled.Bytes(), msg.marshaled.Bytes())
}

func convertToSafePayload(p interface{}) payload.Safe {
	var safePayload payload.Safe
	switch t := p.(type) {
	case uint:
		safePayload = uwrap(t)
	case uint8:
		safePayload = uwrap8(t)
	case uint16:
		safePayload = uwrap16(t)
	case uint32:
		safePayload = uwrap32(t)
	case uint64:
		safePayload = uwrap64(t)
	case int:
		safePayload = iwrap(t)
	case int8:
		safePayload = iwrap8(t)
	case int16:
		safePayload = iwrap16(t)
	case int32:
		safePayload = iwrap32(t)
	case int64:
		safePayload = iwrap64(t)
	case bool:
		safePayload = bwrap(t)
	case string:
		safePayload = swrap(t)
	case *bytes.Buffer:
		safePayload = SafeBuffer{*t}
	case bytes.Buffer:
		safePayload = SafeBuffer{t}
	case payload.Safe:
		safePayload = t
	}

	return safePayload
}

// New creates a new Message.
func New(top topics.Topic, p interface{}) Message {
	safePayload := convertToSafePayload(p)
	return &simple{category: top, payload: safePayload}
}

// NewWithHeader creates a new Message with non-nil header.
func NewWithHeader(t topics.Topic, payload interface{}, header []byte) Message {
	safePayload := convertToSafePayload(payload)
	return &simple{category: t, payload: safePayload, header: header}
}

func (m *simple) initPayloadBuffer(b bytes.Buffer) {
	if m.marshaled == nil {
		m.marshaled = bytes.NewBuffer(b.Bytes())
	}
}

// Unmarshal mutates the buffer by extracting the topic. It create the Message
// by setting the topic and unmarshaling the payload into the proper structure
// It also caches the serialized form within the message.
func Unmarshal(b *bytes.Buffer, h []byte) (Message, error) {
	var err error

	msg := &simple{header: h}
	msg.initPayloadBuffer(*b)

	topic, err := topics.Extract(b)
	if err != nil {
		return nil, err
	}

	msg.category = topic

	switch topic {
	case topics.Block:
		err = UnmarshalBlockMessage(b, msg)
	case topics.GetBlocks:
		err = UnmarshalGetBlocksMessage(b, msg)
	case topics.Inv, topics.GetData:
		err = UnmarshalInvMessage(b, msg)
	case topics.GetCandidate:
		UnmarshalGetCandidateMessage(b, msg)
	case topics.Tx:
		err = UnmarshalTxMessage(b, msg)
	case topics.Candidate:
		err = UnmarshalBlockMessage(b, msg)
	case topics.NewBlock:
		err = UnmarshalNewBlockMessage(b, msg)
	case topics.Reduction:
		err = UnmarshalReductionMessage(b, msg)
	case topics.Agreement:
		err = UnmarshalAgreementMessage(b, msg)
	case topics.Challenge:
		UnmarshalChallengeMessage(b, msg)
	case topics.Response:
		err = UnmarshalResponseMessage(b, msg)
	case topics.Addr:
		UnmarshalAddrMessage(b, msg)
	}

	if err != nil {
		return nil, err
	}

	return *msg, nil
}

// Marshal a Message into a buffer. The buffer *does* include its Category (so
// if this is undesired, the client of this call needs to explicitly call
// topics.Extract on the resulting buffer.
// TODO: interface - once the Gossip preprocessing is removed from the
// Coordinator, there won't be a need for marshalBuffer.
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

// marshal forces the serialization of a Message regardless of its cache.
func marshal(s Message, b *bytes.Buffer) error {
	payload := s.Payload()
	if payload == nil {
		*b = s.Category().ToBuffer()
		return nil
	}

	switch payload := payload.(type) {
	case SafeBuffer:
		*b = payload.Buffer

	default:
		if err := marshalMessage(s.Category(), payload, b); err != nil {
			return err
		}
	}

	return topics.Prepend(b, s.Category())
}

// marshalMessage marshals a Message carrying a payload different than a Buffer.
func marshalMessage(topic topics.Topic, payload interface{}, buf *bytes.Buffer) error {
	var err error

	switch topic {
	case topics.Block:
		blk := payload.(block.Block)
		err = MarshalBlock(buf, &blk)
	case topics.Tx:
		tx := payload.(transactions.ContractCall)
		err = transactions.Marshal(buf, tx)
	case topics.Candidate:
		candidate := payload.(block.Block)
		err = MarshalBlock(buf, &candidate)
	case topics.NewBlock:
		score := payload.(NewBlock)
		err = MarshalNewBlock(buf, score)
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

type uwrap uint

// ConvU converts a Safe into a uint.
func ConvU(p payload.Safe) (uint, error) {
	s, ok := p.(uwrap)
	if !ok {
		return 0, errors.New("payload is not uint")
	}

	return uint(s), nil
}

func (u uwrap) Copy() payload.Safe {
	return u
}

type uwrap8 uint8

// ConvU8 converts a Safe into a uint8.
func ConvU8(p payload.Safe) (uint8, error) {
	s, ok := p.(uwrap8)
	if !ok {
		return 0, errors.New("payload is not a uint8")
	}

	return uint8(s), nil
}

func (u uwrap8) Copy() payload.Safe {
	return u
}

type uwrap16 uint16

// ConvU16 converts a Safe into a uint16.
func ConvU16(p payload.Safe) (uint16, error) {
	s, ok := p.(uwrap16)
	if !ok {
		return 0, errors.New("payload is not a uint16")
	}

	return uint16(s), nil
}

func (u uwrap16) Copy() payload.Safe {
	return u
}

type uwrap32 uint32

// ConvU32 converts a Safe into a uint32.
func ConvU32(p payload.Safe) (uint32, error) {
	s, ok := p.(uwrap32)
	if !ok {
		return 0, errors.New("payload is not a uint32")
	}

	return uint32(s), nil
}

func (u uwrap32) Copy() payload.Safe {
	return u
}

type uwrap64 uint64

// ConvU64 converts a Safe into a uint64.
func ConvU64(p payload.Safe) (uint64, error) {
	s, ok := p.(uwrap64)
	if !ok {
		return 0, errors.New("payload is not a uint64")
	}

	return uint64(s), nil
}

func (u uwrap64) Copy() payload.Safe {
	return u
}

type iwrap int

// ConvI converts a Safe into an int.
func ConvI(p payload.Safe) (int, error) {
	s, ok := p.(iwrap)
	if !ok {
		return 0, errors.New("payload is not a int")
	}

	return int(s), nil
}

func (u iwrap) Copy() payload.Safe {
	return u
}

type iwrap8 int8

// ConvI8 converts a Safe into an int8.
func ConvI8(p payload.Safe) (int8, error) {
	s, ok := p.(iwrap8)
	if !ok {
		return 0, errors.New("payload is not a int8")
	}

	return int8(s), nil
}

func (u iwrap8) Copy() payload.Safe {
	return u
}

type iwrap16 int16

// ConvI16 converts a Safe into an int16.
func ConvI16(p payload.Safe) (int16, error) {
	s, ok := p.(iwrap16)
	if !ok {
		return 0, errors.New("payload is not a int16")
	}

	return int16(s), nil
}

func (u iwrap16) Copy() payload.Safe {
	return u
}

type iwrap32 int32

// ConvI32 converts a Safe into an int32.
func ConvI32(p payload.Safe) (int32, error) {
	s, ok := p.(iwrap32)
	if !ok {
		return 0, errors.New("payload is not a int32")
	}

	return int32(s), nil
}

func (u iwrap32) Copy() payload.Safe {
	return u
}

type iwrap64 int64

// ConvI64 converts a Safe into an int64.
func ConvI64(p payload.Safe) (int64, error) {
	s, ok := p.(iwrap64)
	if !ok {
		return 0, errors.New("payload is not a int64")
	}

	return int64(s), nil
}

func (u iwrap64) Copy() payload.Safe {
	return u
}

type bwrap bool

// ConvBool converts a Safe into a bool.
func ConvBool(p payload.Safe) (bool, error) {
	s, ok := p.(bwrap)
	if !ok {
		return false, errors.New("payload is not a string")
	}

	return bool(s), nil
}

func (u bwrap) Copy() payload.Safe {
	return u
}

type swrap string

// ConvStr converts a Safe into a string.
func ConvStr(p payload.Safe) (string, error) {
	s, ok := p.(swrap)
	if !ok {
		return "", errors.New("payload is not a string")
	}

	return string(s), nil
}

func (u swrap) Copy() payload.Safe {
	return u
}

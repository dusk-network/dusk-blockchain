package agreement

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type CommitteeEvent interface {
	Equal(CommitteeEvent) bool
	BelongsToCommittee(user.Committee) bool
}

type EventDecoder interface {
	Decode(bytes.Buffer) (CommitteeEvent, error)
}

type CommitteeEventCollector interface {
	Contains(CommitteeEvent) bool
	Aggregate(CommitteeEvent) error
}

type AgreementMessage struct {
	VoteSet       []*msg.Vote
	SignedVoteSet []byte
	PubKeyBLS     []byte
	Round         uint64
	Step          uint8
	BlockHash     []byte
}

func (a *AgreementMessage) BelongsToCommittee(committee user.Committee) bool {
	return committee.IsMember(a.PubKeyBLS)
}

func (a *AgreementMessage) Equal(e CommitteeEvent) bool {
	other, ok := e.(*AgreementMessage)
	if !ok {
		return false
	}

	return (bytes.Equal(a.PubKeyBLS, other.PubKeyBLS)) && (a.Round == other.Round) && (a.Step == other.Step)
}

type agreementDecoder struct {
	validate func(*bytes.Buffer) error
}

func newDecoder(validate func(*bytes.Buffer) error) *agreementDecoder {
	return &agreementDecoder{
		validate: validate,
	}
}

func (ad *agreementDecoder) Decode(r bytes.Buffer) (CommitteeEvent, error) {
	if err := ad.validate(&r); err != nil {
		return nil, err
	}

	voteSet, err := msg.DecodeVoteSet(&r)
	if err != nil {
		return nil, err
	}

	var signedVoteSet []byte
	if err := encoding.ReadBLS(&r, &signedVoteSet); err != nil {
		return nil, err
	}

	var pubKeyBLS []byte
	if err := encoding.ReadVarBytes(&r, &pubKeyBLS); err != nil {
		return nil, err
	}

	var blockHash []byte
	if err := encoding.Read256(&r, &blockHash); err != nil {
		return nil, err
	}

	var round uint64
	if err := encoding.ReadUint64(&r, binary.LittleEndian, &round); err != nil {
		return nil, err
	}

	var step uint8
	if err := encoding.ReadUint8(&r, &step); err != nil {
		return nil, err
	}

	decoded := &AgreementMessage{
		VoteSet:       voteSet,
		SignedVoteSet: signedVoteSet,
		PubKeyBLS:     pubKeyBLS,
		BlockHash:     blockHash,
		Round:         round,
		Step:          step,
	}

	return decoded, nil
}

// StepEventCollector is an helper for common operations on stored events
type StepEventCollector map[uint8][]CommitteeEvent

func newStepEventCollector() *StepEventCollector {
	return &StepEventCollector{}
}

func (sec StepEventCollector) Clear() {
	for key := range sec {
		delete(sec, key)
	}
}

// IsDuplicate checks if we already collected this event
func (sec StepEventCollector) IsDuplicate(event CommitteeEvent, step uint8) bool {
	for _, stored := range sec[step] {
		if event.Equal(stored) {
			return true
		}
	}

	return false
}

func (sec StepEventCollector) Store(event CommitteeEvent, step uint8) int {
	eventList := sec[step]
	if eventList == nil {
		// TODO: should this have the Quorum as limit
		eventList = make([]CommitteeEvent, 100)
	}

	// storing the agreement vote for the proper step
	eventList = append(eventList, event)
	sec[step] = eventList
	return len(eventList)
}

type EventCommitteeSubscriber struct {
	eventBus       *wire.EventBus
	msgChan        <-chan *bytes.Buffer
	msgChanID      uint32
	quitChan       <-chan *bytes.Buffer
	quitChanID     uint32
	committeeStore user.Committee
	currentRound   uint64
	decoder        EventDecoder
	collector      CommitteeEventCollector
}

func NewEventCommitteeSubscriber(eventBus *wire.EventBus,
	decoder EventDecoder,
	collector CommitteeEventCollector,
	topic string) *EventCommitteeSubscriber {

	quitChan := make(chan *bytes.Buffer, 1)
	msgChan := make(chan *bytes.Buffer, 100)

	msgChanID := eventBus.Subscribe(topic, msgChan)
	quitChanID := eventBus.Subscribe(string(msg.QuitTopic), msgChan)

	return &EventCommitteeSubscriber{
		eventBus:   eventBus,
		msgChan:    msgChan,
		msgChanID:  msgChanID,
		quitChan:   quitChan,
		quitChanID: quitChanID,
		decoder:    decoder,
	}
}

func (n *EventCommitteeSubscriber) Receive(topic string) {
	for {
		select {
		case <-n.quitChan:
			n.eventBus.Unsubscribe(topic, n.msgChanID)
			n.eventBus.Unsubscribe(string(msg.QuitTopic), n.quitChanID)
			return
		case eventMsg := <-n.msgChan:
			d, err := n.decoder.Decode(*eventMsg)
			if err != nil {
				break
			}
			// casting to an AgreementMessage as we know what Decoder we are using
			n.collector.Aggregate(d)
		}
	}
}

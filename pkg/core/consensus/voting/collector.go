package voting

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	collector struct {
		voteChannel   chan *bytes.Buffer
		unmarshalFunc func(*bytes.Buffer, signer) (wire.Event, error)
		signer        signer
	}

	eventSigner struct {
		*user.Keys
		committee committee.Committee
	}

	signer interface {
		events.ReductionUnmarshaller
		addSignatures(wire.Event) (*bytes.Buffer, error)
		eligibleToVote() bool
	}
)

func newEventSigner(keys *user.Keys, committee committee.Committee) *eventSigner {
	return &eventSigner{
		Keys:      keys,
		committee: committee,
	}
}

func initCollector(broker wire.EventBroker, topic string,
	unmarshalFunc func(*bytes.Buffer, signer) (wire.Event, error),
	signer signer) chan *bytes.Buffer {

	voteChannel := make(chan *bytes.Buffer, 1)
	collector := &collector{
		voteChannel:   voteChannel,
		unmarshalFunc: unmarshalFunc,
		signer:        signer,
	}
	go wire.NewTopicListener(broker, collector, topic).Accept()
	return voteChannel
}

func (c *collector) createVote(ev wire.Event) *bytes.Buffer {
	if !c.signer.eligibleToVote() {
		return nil
	}

	buffer, _ := c.signer.addSignatures(ev)
	return buffer
}

func (c *collector) Collect(r *bytes.Buffer) error {
	info, err := c.unmarshalFunc(r, c.signer)
	if err != nil {
		return err
	}

	c.voteChannel <- c.createVote(info)
	return nil
}

func unmarshalBlockReduction(reductionBuffer *bytes.Buffer, signer signer) (wire.Event, error) {
	var round uint64
	if err := encoding.ReadUint64(reductionBuffer, binary.LittleEndian, &round); err != nil {
		return nil, err
	}

	var step uint8
	if err := encoding.ReadUint8(reductionBuffer, &step); err != nil {
		return nil, err
	}

	var votedHash []byte
	if err := encoding.Read256(reductionBuffer, &votedHash); err != nil {
		return nil, err
	}

	return &events.Reduction{
		Header: &events.Header{
			Round: round,
			Step:  step,
		},
		VotedHash: votedHash,
	}, nil
}

func unmarshalBlockAgreement(agreementBuffer *bytes.Buffer, signer signer) (wire.Event, error) {
	var round uint64
	if err := encoding.ReadUint64(agreementBuffer, binary.LittleEndian, &round); err != nil {
		return nil, err
	}

	var step uint8
	if err := encoding.ReadUint8(agreementBuffer, &step); err != nil {
		return nil, err
	}

	var agreedHash []byte
	if err := encoding.Read256(agreementBuffer, &agreedHash); err != nil {
		return nil, err
	}

	voteSet, err := signer.UnmarshalVoteSet(agreementBuffer)
	if err != nil {
		return nil, err
	}

	return &events.Agreement{
		Header: &events.Header{
			Round: round,
			Step:  step,
		},
		AgreedHash: agreedHash,
		VoteSet:    voteSet,
	}, nil
}

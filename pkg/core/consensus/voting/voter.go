package voting

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Voter is responsible for signing and sending out consensus related
// voting messages to the wire.
type (
	Voter struct {
		eventBus *wire.EventBus
		*consensus.EventHeaderMarshaller

		reductionChannel   chan *reduction
		agreementChannel   chan *agreement
		phaseUpdateChannel chan []byte
		roundUpdateChannel chan uint64

		winningBlockHash []byte
		*user.Keys
		committee committee.Committee
	}
	reduction struct {
		round     uint64
		step      uint8
		votedHash []byte
	}

	reductionCollector struct {
		reductionChannel chan *reduction
	}

	agreement struct {
		round   uint64
		step    uint8
		hash    []byte
		voteSet []*msg.Vote
	}

	agreementCollector struct {
		agreementChannel chan *agreement
	}
)

// NewVoter will return an initialized Voter struct.
func NewVoter(eventBus *wire.EventBus, keys *user.Keys,
	committee committee.Committee) *Voter {

	reductionChannel := make(chan *reduction, 2)
	agreementChannel := make(chan *agreement, 1)

	reductionCollector := &reductionCollector{reductionChannel}
	wire.NewEventSubscriber(eventBus, reductionCollector,
		string(msg.OutgoingReductionTopic)).Accept()

	agreementCollector := &agreementCollector{agreementChannel}
	wire.NewEventSubscriber(eventBus, agreementCollector,
		msg.OutgoingAgreementTopic).Accept()

	phaseUpdateCollector := consensus.InitPhaseCollector(eventBus)
	roundUpdateCollector := consensus.InitRoundCollector(eventBus)

	return &Voter{
		eventBus:              eventBus,
		EventHeaderMarshaller: &consensus.EventHeaderMarshaller{},
		reductionChannel:      reductionChannel,
		agreementChannel:      agreementChannel,
		phaseUpdateChannel:    phaseUpdateCollector.BlockHashChan,
		roundUpdateChannel:    roundUpdateCollector.RoundChan,
		Keys:                  keys,
		committee:             committee,
	}
}

func (v Voter) addPubKeyAndSig(message *bytes.Buffer,
	signature []byte) (*bytes.Buffer, error) {

	buffer := bytes.NewBuffer([]byte(*v.EdPubKey))
	if err := encoding.Write512(buffer, signature); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(message.Bytes()); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (v Voter) eligibleToVote() bool {
	return v.committee.IsMember(v.Keys.BLSPubKey.Marshal())
}

// Listen will set the Voter up to listen for incoming requests to vote.
func (v Voter) Listen() {
	for {
		select {
		case blockHash := <-v.phaseUpdateChannel:
			v.winningBlockHash = blockHash
		case <-v.roundUpdateChannel:
			v.winningBlockHash = nil
		case r := <-v.reductionChannel:
			if v.eligibleToVote() {
				if err := v.voteReduction(r); err != nil {
					// Log
					return
				}
			}
		case a := <-v.agreementChannel:
			if v.eligibleToVote() {
				if err := v.voteAgreement(a); err != nil {
					// Log
					return
				}
			}
		}
	}
}

func (r *reductionCollector) Collect(reductionBuffer *bytes.Buffer) error {
	reduction, err := unmarshalReduction(reductionBuffer)
	if err != nil {
		return err
	}

	r.reductionChannel <- reduction
	return nil
}

func (a *agreementCollector) Collect(agreementBuffer *bytes.Buffer) error {
	agreement, err := unmarshalAgreement(agreementBuffer)
	if err != nil {
		return err
	}

	a.agreementChannel <- agreement
	return nil
}

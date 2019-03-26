package voting

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// Voter is responsible for signing and sending out consensus related
// voting messages to the wire.
type Voter struct {
	eventBus               *wire.EventBus
	blockReductionChannel  chan *bytes.Buffer
	blockAgreementChannel  chan *bytes.Buffer
	sigSetReductionChannel chan *bytes.Buffer
	sigSetAgreementChannel chan *bytes.Buffer
}

// NewVoter will return an initialized Voter struct.
func NewVoter(eventBus *wire.EventBus, keys *user.Keys, committee committee.Committee) *Voter {

	blockReductionChannel := initCollector(eventBus, msg.OutgoingBlockReductionTopic,
		unmarshalBlockReduction, newBlockReductionSigner(keys, committee))
	sigSetReductionChannel := initCollector(eventBus, msg.OutgoingSigSetReductionTopic,
		unmarshalSigSetReduction, newSigSetReductionSigner(keys, committee))
	blockAgreementChannel := initCollector(eventBus, msg.OutgoingBlockAgreementTopic,
		unmarshalBlockAgreement, newBlockAgreementSigner(keys, committee))
	sigSetAgreementChannel := initCollector(eventBus, msg.OutgoingSigSetAgreementTopic,
		unmarshalSigSetAgreement, newSigSetAgreementSigner(keys, committee))

	return &Voter{
		eventBus:               eventBus,
		blockReductionChannel:  blockReductionChannel,
		sigSetReductionChannel: sigSetReductionChannel,
		blockAgreementChannel:  blockAgreementChannel,
		sigSetAgreementChannel: sigSetAgreementChannel,
	}
}

// Listen will set the Voter up to listen for incoming requests to vote.
// TODO: create topics that the peer manager listens to. currently using
// the topics that are normally associated with incoming messages.
func (v *Voter) Listen() {
	for {
		select {
		case m := <-v.blockReductionChannel:
			v.eventBus.Publish(string(topics.BlockReduction), m)
		case m := <-v.sigSetReductionChannel:
			v.eventBus.Publish(string(topics.SigSetReduction), m)
		case m := <-v.blockAgreementChannel:
			v.eventBus.Publish(string(topics.BlockAgreement), m)
		case m := <-v.sigSetAgreementChannel:
			v.eventBus.Publish(string(topics.SigSetAgreement), m)
		}
	}
}

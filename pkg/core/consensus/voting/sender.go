package voting

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func LaunchVotingComponent(eventBus *wire.EventBus, keys *user.Keys,
	committee committee.Committee) *sender {

	sender := newSender(eventBus, keys, committee)
	go sender.Listen()
	return sender
}

// sender is responsible for signing and sending out consensus related
// voting messages to the wire.
type sender struct {
	eventBus               *wire.EventBus
	blockReductionChannel  chan *bytes.Buffer
	blockAgreementChannel  chan *bytes.Buffer
	sigSetReductionChannel chan *bytes.Buffer
	sigSetAgreementChannel chan *bytes.Buffer
}

// newSender will return an initialized sender struct.
func newSender(eventBus *wire.EventBus, keys *user.Keys, committee committee.Committee) *sender {
	blockReductionChannel := initCollector(eventBus, msg.OutgoingBlockReductionTopic,
		unmarshalBlockReduction, newBlockReductionSigner(keys, committee))
	sigSetReductionChannel := initCollector(eventBus, msg.OutgoingSigSetReductionTopic,
		unmarshalSigSetReduction, newSigSetReductionSigner(keys, committee))
	blockAgreementChannel := initCollector(eventBus, msg.OutgoingBlockAgreementTopic,
		unmarshalBlockAgreement, newBlockAgreementSigner(keys, committee))
	sigSetAgreementChannel := initCollector(eventBus, msg.OutgoingSigSetAgreementTopic,
		unmarshalSigSetAgreement, newSigSetAgreementSigner(keys, committee))

	return &sender{
		eventBus:               eventBus,
		blockReductionChannel:  blockReductionChannel,
		sigSetReductionChannel: sigSetReductionChannel,
		blockAgreementChannel:  blockAgreementChannel,
		sigSetAgreementChannel: sigSetAgreementChannel,
	}
}

// Listen will set the sender up to listen for incoming requests to vote.
func (v *sender) Listen() {
	for {
		select {
		case m := <-v.blockReductionChannel:
			v.eventBus.Publish(string(topics.PeerBlockReduction), m)
		case m := <-v.sigSetReductionChannel:
			v.eventBus.Publish(string(topics.PeerSigSetReduction), m)
		case m := <-v.blockAgreementChannel:
			v.eventBus.Publish(string(topics.PeerBlockAgreement), m)
		case m := <-v.sigSetAgreementChannel:
			v.eventBus.Publish(string(topics.PeerSigSetAgreement), m)
		}
	}
}

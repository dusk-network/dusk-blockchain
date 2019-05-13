package voting

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// LaunchVotingComponent creates the collectors appointed to sign reduction and agreement messages.
func LaunchVotingComponent(broker wire.EventBroker, committee committee.Committee, keys *user.Keys) {
	reductionCollector := NewReductionSigner(keys, broker)
	go wire.NewTopicListener(broker, reductionCollector, msg.OutgoingBlockReductionTopic).Accept()

	agreementCollector := NewAgreementSigner(committee, keys, broker)
	go wire.NewTopicListener(broker, agreementCollector, msg.OutgoingBlockAgreementTopic).Accept()
}

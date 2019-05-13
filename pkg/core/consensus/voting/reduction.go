package voting

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type reductionSigner struct {
	*user.Keys
	publisher wire.EventPublisher
}

func NewReductionSigner(keys *user.Keys, p wire.EventPublisher) *reductionSigner {
	return &reductionSigner{
		publisher: p,
		Keys:      keys,
	}
}

func (bs *reductionSigner) Collect(buf *bytes.Buffer) error {
	if err := events.SignReduction(buf, bs.Keys); err != nil {
		return err
	}

	message, err := wire.AddTopic(buf, topics.Reduction)
	if err != nil {
		return err
	}
	bs.publisher.Publish(string(topics.Gossip), message)
	return nil
}

package voting

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	collector struct {
		voteChannel chan *bytes.Buffer
		signer      signer
	}

	eventSigner struct {
		*user.Keys
	}

	signer interface {
		events.ReductionUnmarshaller
		AddSignatures(wire.Event) (*bytes.Buffer, error)
	}
)

func newEventSigner(keys *user.Keys) *eventSigner {
	return &eventSigner{
		Keys: keys,
	}
}

func initCollector(broker wire.EventBroker, topic string, signer signer) chan *bytes.Buffer {

	voteChannel := make(chan *bytes.Buffer, 1)
	collector := &collector{
		voteChannel: voteChannel,
		signer:      signer,
	}
	go wire.NewTopicListener(broker, collector, topic).Accept()
	return voteChannel
}

func (c *collector) createVote(ev wire.Event) *bytes.Buffer {
	buffer, _ := c.signer.AddSignatures(ev)
	return buffer
}

func (c *collector) Collect(r *bytes.Buffer) error {
	// copy shared pointer
	copyBuf := *r
	ev := c.signer.NewEvent()
	if err := c.signer.Unmarshal(&copyBuf, ev); err != nil {
		return err
	}

	c.voteChannel <- c.createVote(ev)
	return nil
}

package consensus

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/v2/block"
)

// roundCollector is a simple wrapper over a channel to get round notifications.
// It is not supposed to be used directly. Components interestesd in Round updates
// should use InitRoundUpdate instead
type (
	roundCollector struct {
		roundChan chan RoundUpdate
	}

	acceptedBlockCollector struct {
		blockChan chan<- block.Block
	}

	RoundUpdate struct {
		Round   uint64
		P       user.Provisioners
		BidList user.BidList
		Seed    []byte
		Hash    []byte
	}
)

// InitRoundUpdate initializes a Round update channel and fires up the TopicListener
// as well. Its purpose is to lighten up a bit the amount of arguments in creating
// the handler for the collectors. Also it removes the need to store subscribers on
// the consensus process
func InitRoundUpdate(subscriber eventbus.Subscriber) <-chan RoundUpdate {
	roundChan := make(chan RoundUpdate, 1)
	roundCollector := &roundCollector{roundChan}
	l := eventbus.NewCallbackListener(roundCollector.Collect)
	subscriber.Subscribe(topics.RoundUpdate, l)
	return roundChan
}

// Collect as specified in the EventCollector interface. In this case Collect simply
// performs unmarshaling of the round event
func (r *roundCollector) Collect(m message.Message) error {
	update := m.Payload().(RoundUpdate)
	r.roundChan <- update
	return nil
}

// InitAcceptedBlockUpdate init listener to get updates about lastly accepted block in the chain
func InitAcceptedBlockUpdate(subscriber eventbus.Subscriber) (chan block.Block, uint32) {
	acceptedBlockChan := make(chan block.Block)
	collector := &acceptedBlockCollector{acceptedBlockChan}
	l := eventbus.NewCallbackListener(collector.Collect)
	id := subscriber.Subscribe(topics.AcceptedBlock, l)
	return acceptedBlockChan, id
}

// Collect as defined in the EventCollector interface. It reconstructs the bidList and notifies about it
func (c *acceptedBlockCollector) Collect(m message.Message) error {
	b := m.Payload().(block.Block)
	c.blockChan <- b
	return nil
}

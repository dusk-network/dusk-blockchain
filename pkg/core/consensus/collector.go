package consensus

import (
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
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

	// RoundUpdate carries the data about the new Round, such as the active
	// Provisioners, the BidList, the Seed and the Hash
	RoundUpdate struct {
		Round uint64
		P     user.Provisioners
		Seed  []byte
		Hash  []byte
	}
)

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers
func (r RoundUpdate) Copy() payload.Safe {
	ru := RoundUpdate{
		Round: r.Round,
		P:     r.P.Copy(),
		Seed:  make([]byte, len(r.Seed)),
		Hash:  make([]byte, len(r.Hash)),
	}

	copy(ru.Seed, r.Seed)
	copy(ru.Hash, r.Hash)
	return ru
}

// InitRoundUpdate initializes a Round update channel and fires up the TopicListener
// as well. Its purpose is to lighten up a bit the amount of arguments in creating
// the handler for the collectors. Also it removes the need to store subscribers on
// the consensus process
func InitRoundUpdate(subscriber eventbus.Subscriber) <-chan RoundUpdate {
	roundChan := make(chan RoundUpdate, 1)
	roundCollector := &roundCollector{roundChan}
	collectListener := eventbus.NewCallbackListener(roundCollector.Collect)
	if config.Get().General.SafeCallbackListener {
		collectListener = eventbus.NewSafeCallbackListener(roundCollector.Collect)
	}
	subscriber.Subscribe(topics.RoundUpdate, collectListener)
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
	collectListener := eventbus.NewCallbackListener(collector.Collect)
	if config.Get().General.SafeCallbackListener {
		collectListener = eventbus.NewSafeCallbackListener(collector.Collect)
	}
	id := subscriber.Subscribe(topics.AcceptedBlock, collectListener)
	return acceptedBlockChan, id
}

// Collect as defined in the EventCollector interface. It reconstructs the bidList and notifies about it
func (c *acceptedBlockCollector) Collect(m message.Message) error {
	b := m.Payload().(block.Block)
	c.blockChan <- b
	return nil
}

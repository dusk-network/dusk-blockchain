package consensus

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
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

	regenerationCollector struct {
		regenerationChan chan AsyncState
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

// UpdateRound is a shortcut for propagating a round
func UpdateRound(bus eventbus.Publisher, round uint64) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, round)
	bus.Publish(topics.RoundUpdate, bytes.NewBuffer(b))
}

// InitRoundUpdate initializes a Round update channel and fires up the TopicListener
// as well. Its purpose is to lighten up a bit the amount of arguments in creating
// the handler for the collectors. Also it removes the need to store subscribers on
// the consensus process
func InitRoundUpdate(subscriber eventbus.Subscriber) <-chan RoundUpdate {
	roundChan := make(chan RoundUpdate, 1)
	roundCollector := &roundCollector{roundChan}
	eventbus.NewTopicListener(subscriber, roundCollector, topics.RoundUpdate, eventbus.ChannelType)
	return roundChan
}

// Collect as specified in the EventCollector interface. In this case Collect simply
// performs unmarshalling of the round event
func (r *roundCollector) Collect(roundBuffer bytes.Buffer) error {
	var round uint64
	rb := &roundBuffer
	if err := encoding.ReadUint64LE(rb, &round); err != nil {
		return err
	}

	provisioners, err := user.UnmarshalProvisioners(rb)
	if err != nil {
		return err
	}

	bidList, err := user.UnmarshalBidList(rb)
	if err != nil {
		return err
	}

	seed := make([]byte, 33)
	if err := encoding.ReadBLS(rb, seed); err != nil {
		return err
	}

	hash := make([]byte, 32)
	if err := encoding.Read256(rb, hash); err != nil {
		return err
	}

	r.roundChan <- RoundUpdate{round, provisioners, bidList, seed, hash}
	return nil
}

// InitBlockRegenerationCollector initializes a regeneration channel, creates a
// regenerationCollector, and subscribes this collector to the BlockRegenerationTopic.
// The channel is then returned.
func InitBlockRegenerationCollector(subscriber eventbus.Subscriber) chan AsyncState {
	regenerationChan := make(chan AsyncState, 1)
	collector := &regenerationCollector{regenerationChan}
	eventbus.NewTopicListener(subscriber, collector, topics.BlockRegeneration, eventbus.ChannelType)
	return regenerationChan
}

func (rg *regenerationCollector) Collect(r bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(r.Bytes()[:8])
	step := uint8(r.Bytes()[8])
	state := AsyncState{
		Round: round,
		Step:  step,
	}
	rg.regenerationChan <- state
	return nil
}

// InitAcceptedBlockUpdate init listener to get updates about lastly accepted block in the chain
func InitAcceptedBlockUpdate(subscriber eventbus.Subscriber) (chan block.Block, eventbus.TopicListener) {
	acceptedBlockChan := make(chan block.Block)
	collector := &acceptedBlockCollector{acceptedBlockChan}
	tl := eventbus.NewTopicListener(subscriber, collector, topics.AcceptedBlock, eventbus.ChannelType)
	return acceptedBlockChan, tl
}

// Collect as defined in the EventCollector interface. It reconstructs the bidList and notifies about it
func (c *acceptedBlockCollector) Collect(r bytes.Buffer) error {
	b := block.NewBlock()
	if err := block.Unmarshal(&r, b); err != nil {
		return err
	}

	c.blockChan <- *b
	return nil
}

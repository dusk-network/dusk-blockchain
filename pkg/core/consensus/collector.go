package consensus

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
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
		P       user.Stakers
		BidList user.BidList
	}
)

// UpdateRound is a shortcut for propagating a round
func UpdateRound(bus wire.EventPublisher, round uint64) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, round)
	bus.Publish(msg.RoundUpdateTopic, bytes.NewBuffer(b))
}

// InitRoundUpdate initializes a Round update channel and fires up the TopicListener
// as well. Its purpose is to lighten up a bit the amount of arguments in creating
// the handler for the collectors. Also it removes the need to store subscribers on
// the consensus process
func InitRoundUpdate(subscriber wire.EventSubscriber) <-chan RoundUpdate {
	roundChan := make(chan RoundUpdate, 1)
	roundCollector := &roundCollector{roundChan}
	go wire.NewTopicListener(subscriber, roundCollector, string(msg.RoundUpdateTopic)).Accept()
	return roundChan
}

// Collect as specified in the EventCollector interface. In this case Collect simply
// performs unmarshalling of the round event
func (r *roundCollector) Collect(roundBuffer *bytes.Buffer) error {
	var round uint64
	if err := encoding.ReadUint64(roundBuffer, binary.LittleEndian, &round); err != nil {
		return err
	}

	stakers, err := user.UnmarshalStakers(roundBuffer)
	if err != nil {
		return err
	}

	bidList, err := user.UnmarshalBidList(roundBuffer)
	if err != nil {
		return err
	}

	r.roundChan <- RoundUpdate{round, stakers, bidList}
	return nil
}

// InitBlockRegenerationCollector initializes a regeneration channel, creates a
// regenerationCollector, and subscribes this collector to the BlockRegenerationTopic.
// The channel is then returned.
func InitBlockRegenerationCollector(subscriber wire.EventSubscriber) chan AsyncState {
	regenerationChan := make(chan AsyncState, 1)
	collector := &regenerationCollector{regenerationChan}
	go wire.NewTopicListener(subscriber, collector, msg.BlockRegenerationTopic).Accept()
	return regenerationChan
}

func (rg *regenerationCollector) Collect(r *bytes.Buffer) error {
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
func InitAcceptedBlockUpdate(subscriber wire.EventSubscriber) (chan block.Block, *wire.TopicListener) {
	acceptedBlockChan := make(chan block.Block)
	collector := &acceptedBlockCollector{acceptedBlockChan}
	tl := wire.NewTopicListener(subscriber, collector, string(topics.AcceptedBlock))
	go tl.Accept()
	return acceptedBlockChan, tl
}

// Collect as defined in the EventCollector interface. It reconstructs the bidList and notifies about it
func (c *acceptedBlockCollector) Collect(r *bytes.Buffer) error {
	b := block.NewBlock()
	if err := block.Unmarshal(r, b); err != nil {
		return err
	}

	c.blockChan <- *b
	return nil
}

package consensus

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
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
		roundChan chan uint64
	}

	regenerationCollector struct {
		regenerationChan chan AsyncState
	}

	bidListCollector struct {
		bidListChan chan<- user.Bid
	}

	acceptedBlockCollector struct {
		blockChan chan<- block.Block
	}
)

// UpdateRound is a shortcut for propagating a round
func UpdateRound(bus eventbus.Publisher, round uint64) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, round)
	bus.Publish(msg.RoundUpdateTopic, bytes.NewBuffer(b))
}

// InitRoundUpdate initializes a Round update channel and fires up the TopicListener
// as well. Its purpose is to lighten up a bit the amount of arguments in creating
// the handler for the collectors. Also it removes the need to store subscribers on
// the consensus process
func InitRoundUpdate(subscriber eventbus.Subscriber) <-chan uint64 {
	roundChan := make(chan uint64, 1)
	roundCollector := &roundCollector{roundChan}
	go eventbus.NewTopicListener(subscriber, roundCollector, string(msg.RoundUpdateTopic)).Accept()
	return roundChan
}

// Collect as specified in the EventCollector interface. In this case Collect simply
// performs unmarshalling of the round event
func (r *roundCollector) Collect(roundBuffer *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	r.roundChan <- round
	return nil
}

// InitBlockRegenerationCollector initializes a regeneration channel, creates a
// regenerationCollector, and subscribes this collector to the BlockRegenerationTopic.
// The channel is then returned.
func InitBlockRegenerationCollector(subscriber eventbus.Subscriber) chan AsyncState {
	regenerationChan := make(chan AsyncState, 1)
	collector := &regenerationCollector{regenerationChan}
	go eventbus.NewTopicListener(subscriber, collector, msg.BlockRegenerationTopic).Accept()
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

// InitBidListUpdate creates and initiates a channel for the updates in the BidList
func InitBidListUpdate(subscriber eventbus.Subscriber) chan user.Bid {
	bidListChan := make(chan user.Bid)
	collector := &bidListCollector{bidListChan}
	go eventbus.NewTopicListener(subscriber, collector, string(msg.BidListTopic)).Accept()
	return bidListChan
}

// Collect implements EventCollector.
// It reconstructs the bidList and sends it on its BidListChan
func (b *bidListCollector) Collect(r *bytes.Buffer) error {
	b.bidListChan <- decodeBid(r)
	return nil
}

func decodeBid(r *bytes.Buffer) user.Bid {
	var xSlice []byte
	if err := encoding.Read256(r, &xSlice); err != nil {
		panic(err)
	}

	var mSlice []byte
	if err := encoding.Read256(r, &mSlice); err != nil {
		panic(err)
	}

	var endHeight uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &endHeight); err != nil {
		panic(err)
	}

	var x [32]byte
	copy(x[:], xSlice)
	var m [32]byte
	copy(m[:], mSlice)
	return user.Bid{x, m, endHeight}
}

// InitAcceptedBlockUpdate init listener to get updates about lastly accepted block in the chain
func InitAcceptedBlockUpdate(subscriber eventbus.Subscriber) (chan block.Block, *eventbus.TopicListener) {
	acceptedBlockChan := make(chan block.Block)
	collector := &acceptedBlockCollector{acceptedBlockChan}
	tl := eventbus.NewTopicListener(subscriber, collector, string(topics.AcceptedBlock))
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

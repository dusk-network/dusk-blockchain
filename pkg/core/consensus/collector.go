package consensus

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/block"
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

func DecodeRound(rb *bytes.Buffer, update *RoundUpdate) error {
	var round uint64
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

	update.P = provisioners
	update.Round = round
	update.BidList = bidList
	update.Seed = seed
	update.Hash = hash
	return nil
}

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
// performs unmarshalling of the round event
func (r *roundCollector) Collect(roundBuffer bytes.Buffer) error {
	update := RoundUpdate{}
	if err := DecodeRound(&roundBuffer, &update); err != nil {
		return err
	}
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
func (c *acceptedBlockCollector) Collect(r bytes.Buffer) error {
	b := block.NewBlock()
	if err := message.UnmarshalBlock(&r, b); err != nil {
		return err
	}

	c.blockChan <- *b
	return nil
}

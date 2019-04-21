package consensus

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
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
		BidListChan chan<- user.BidList
	}
)

// InitRoundUpdate initializes a Round update channel and fires up the TopicListener as well.
// Its purpose is to lighten up a bit the amount of arguments in creating the handler for the collectors. Also it removes the need to store subscribers on the consensus process
func InitRoundUpdate(subscriber wire.EventSubscriber) chan uint64 {
	roundChan := make(chan uint64, 1)
	roundCollector := &roundCollector{roundChan}
	go wire.NewTopicListener(subscriber, roundCollector, string(msg.RoundUpdateTopic)).Accept()
	return roundChan
}

// Collect as specified in the EventCollector interface. In this case Collect simply performs unmarshalling of the round event
func (r *roundCollector) Collect(roundBuffer *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	r.roundChan <- round
	return nil
}

func InitBlockRegenerationCollector(subscriber wire.EventSubscriber) chan AsyncState {
	regenerationChan := make(chan AsyncState, 1)
	collector := &regenerationCollector{regenerationChan}
	go wire.NewTopicListener(subscriber, collector, msg.BlockRegenerationTopic).Accept()
	return regenerationChan
}

func (sc *regenerationCollector) Collect(r *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(r.Bytes()[:8])
	step := uint8(r.Bytes()[8])
	state := AsyncState{
		Round: round,
		Step:  step,
	}
	sc.regenerationChan <- state
	return nil
}

// InitBidListUpdate creates and initiates a channel for the updates in the BidList
func InitBidListUpdate(subscriber wire.EventSubscriber) chan user.BidList {
	bidListChan := make(chan user.BidList)
	collector := &bidListCollector{bidListChan}
	go wire.NewTopicListener(subscriber, collector, string(msg.BidListTopic)).Accept()
	return bidListChan
}

// Collect as defined in the EventCollector interface. It reconstructs the bidList and notifies about it
func (l *bidListCollector) Collect(r *bytes.Buffer) error {
	bidList, err := user.ReconstructBidListSubset(r.Bytes())
	if err != nil {
		return nil
	}
	l.BidListChan <- bidList
	return nil
}

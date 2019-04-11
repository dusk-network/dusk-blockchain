package consensus

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	// EventHandler encapsulate logic specific to the various Collectors.
	//Each Collector needs to verify, prioritize and extract information from Events.
	//EventHandler is the interface that abstracts these operations away.
	//The implementors of this interface is the real differentiator of the various consensus components
	EventHandler interface {
		wire.EventVerifier
		wire.EventPrioritizer
		wire.EventUnMarshaller
		NewEvent() wire.Event
		ExtractHeader(wire.Event, *EventHeader)
	}

	// roundCollector is a simple wrapper over a channel to get round notifications. It is not supposed to be used directly. Components interestesd in Round updates should use InitRoundUpdate instead
	roundCollector struct {
		roundChan chan uint64
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

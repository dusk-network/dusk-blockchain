package consensus

import (
	"bytes"
	"encoding/binary"

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

	// phaseCollector is not supposed to be used directly. Components interested in Phase Updates should import InitPhaseUpdate instead
	phaseCollector struct {
		blockHashChan chan []byte
	}

	// roundCollector is a simple wrapper over a channel to get round notifications. It is not supposed to be used directly. Components interestesd in Round updates should use InitRoundUpdate instead
	roundCollector struct {
		roundChan chan uint64
	}
)

// InitRoundUpdate initializes a Round update channel and fires up the EventSubscriber as well.
// Its purpose is to lighten up a bit the amount of arguments in creating the handler for the collectors. Also it removes the need to store subscribers on the consensus process
func InitRoundUpdate(eventBus *wire.EventBus) chan uint64 {
	roundChan := make(chan uint64, 1)
	roundCollector := &roundCollector{roundChan}
	go wire.NewEventSubscriber(eventBus, roundCollector, string(msg.RoundUpdateTopic)).Accept()
	return roundChan
}

// Collect as specified in the EventCollector interface.
//In this case Collect simply performs unmarshalling of the round event
func (r *roundCollector) Collect(roundBuffer *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	r.roundChan <- round
	return nil
}

// InitPhaseUpdate initializes a channel for blockHash and fires up the EventSubscriber as well.
// Its purpose is to lighten up a bit the amount of arguments in creating the handler for the collectors. Also it removes the need to store subscribers on the consensus process
func InitPhaseUpdate(eventBus *wire.EventBus) chan []byte {
	phaseUpdateChan := make(chan []byte)
	collector := &phaseCollector{phaseUpdateChan}
	go wire.NewEventSubscriber(eventBus, collector, string(msg.PhaseUpdateTopic)).Accept()
	return phaseUpdateChan
}

func (p *phaseCollector) Collect(phaseBuffer *bytes.Buffer) error {
	p.blockHashChan <- phaseBuffer.Bytes()
	return nil
}

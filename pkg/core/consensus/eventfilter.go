package consensus

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
)

type (
	// EventHandler encapsulates logic specific to the various EventFilters.
	// Each EventFilter needs to verify, prioritize and extract information from Events.
	// EventHandler is the interface that abstracts these operations away.
	// The implementors of this interface is the real differentiator of the various
	// consensus components
	EventHandler interface {
		wire.EventVerifier
		wire.EventMarshaller
		wire.EventDeserializer
		ExtractHeader(wire.Event) *header.Header
	}
)

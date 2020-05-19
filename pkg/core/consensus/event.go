package consensus

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
)

var emptyHash [32]byte

// InternalPacket is a specialization of the Payload of message.Message. It is used to
// unify messages used by the consensus, which need to carry the header.Header
// for consensus specific operations
type InternalPacket interface {
	payload.Safe
	State() header.Header
}

type empty struct{}

// State returns an empty Header
func (e empty) State() header.Header {
	return header.Header{}
}

// EmptyPacket returns an empty InternalPacket
func EmptyPacket() InternalPacket {
	return empty{}
}

// Copy is a noop
func (e empty) Copy() payload.Safe {
	return empty{}
}

// Packet is a consensus message payload with a full Header
type Packet interface {
	InternalPacket
	Sender() []byte
}

// PacketFactory is used by the signer/coordinator to create internal
// messages
type PacketFactory interface {
	Create([]byte, uint64, uint8) InternalPacket
}

// Restarter creates the Restart message used by the Generator and the
// Reduction
type Restarter struct{}

// Copy is a no-op on the Restarter since it does not carry any content
func (r Restarter) Copy() payload.Safe {
	return Restarter{}
}

// Create a Restart message to restart the consensus
func (r Restarter) Create(sender []byte, round uint64, step uint8) InternalPacket {
	return header.Header{
		Round:     round,
		Step:      step,
		BlockHash: emptyHash[:],
		PubKeyBLS: sender,
	}
}

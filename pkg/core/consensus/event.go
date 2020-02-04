package consensus

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
)

var emptyHash [32]byte

// Message is a specialization of the Payload of message.Message. It is used to
// unify messages used by the consensus, which need to carry the header.Header
// for consensus specific operations
// TODO: interface - consider breaking the Header down into
// AsyncState/SignedState/Header
type InternalPacket interface {
	State() header.Header
}

type empty struct{}

func (e empty) State() header.Header {
	return header.Header{}
}

func EmptyPacket() InternalPacket {
	return empty{}
}

// Packet is a consensus message payload with a full Header
type Packet interface {
	InternalPacket
	Sender() []byte
}

// InternalMsgFactory is used by the signer/coordinator to create internal
// messages
type PacketFactory interface {
	Create([]byte, uint64, uint8) InternalPacket
}

// Restarter creates the Restart message used by the Generator and the
// Reduction
type Restarter struct{}

func (r Restarter) Create(sender []byte, round uint64, step uint8) InternalPacket {
	return header.Header{
		Round:     round,
		Step:      step,
		BlockHash: emptyHash[:],
		PubKeyBLS: sender,
	}
}

package message

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
)

// Initialization carries the keys needed to start consensus.
type Initialization struct {
	PublicKey *keys.PublicKey
	BLSKeys   *key.Keys
}

// NewInitialization returns a populated Initialization message.
func NewInitialization(pk *keys.PublicKey, blsKeys *key.Keys) Initialization {
	return Initialization{pk, blsKeys}
}

// Copy an Initialization message.
// Implements the payload.Safe interface.
// TODO: implement
func (i Initialization) Copy() payload.Safe {
	return i
}

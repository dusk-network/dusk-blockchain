package key

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
)

type WalletKeys struct {
	PublicKey     *keys.PublicKey
	ConsensusKeys Keys
}

func (r WalletKeys) Copy() payload.Safe {
	w := WalletKeys{
		PublicKey: r.PublicKey.Copy(),
		/// TODO: deep copy
		ConsensusKeys: r.ConsensusKeys,
	}

	return w
}

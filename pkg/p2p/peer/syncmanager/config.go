package syncmanager

import "gitlab.dusk.network/dusk-core/dusk-go/pkg/blockchain"

type Config struct {
	Chain    *blockchain.Chain
	BestHash []byte
}

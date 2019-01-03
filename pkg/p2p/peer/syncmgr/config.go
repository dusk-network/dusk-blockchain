package syncmgr

import "gitlab.dusk.network/dusk-core/dusk-go/pkg/blockchain"

// Config holds a pointer to the block chain and its latest hash
type Config struct {
	Chain    *blockchain.Chain
	BestHash []byte
}

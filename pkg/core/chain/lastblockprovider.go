package chain

import (
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// lastBlockProvider stores and provides blockchain tip of the local chain state
// Last Block will be the same with network tip only if the node is fully synced
type lastBlockProvider struct {
	mu        sync.RWMutex
	lastBlock payload.Safe

	reqChan <-chan rpcbus.Request
}

func newLastBlockProvider(rpcBus *rpcbus.RPCBus, b *block.Block) (*lastBlockProvider, error) {

	// set up rpcbus channels
	reqChan := make(chan rpcbus.Request, 1)
	if err := rpcBus.Register(topics.GetLastBlock, reqChan); err != nil {
		return nil, err
	}

	p := &lastBlockProvider{
		reqChan:   reqChan,
		lastBlock: nil,
	}

	p.Set(b)

	go p.Listen()
	return p, nil
}

// Set updates last block with a full copy of the new block
func (p *lastBlockProvider) Set(n *block.Block) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if n != nil {
		p.lastBlock = n.Copy()
	} else {
		// reset payload
		p.lastBlock = nil
	}
}

// Get returns a safe copy of last block
func (p *lastBlockProvider) Get() block.Block {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.lastBlock == nil {
		return *block.NewBlock()
	}

	return p.lastBlock.(block.Block)
}

// Listen provides response to the topics.GetLastBlock rpcbus call
func (p *lastBlockProvider) Listen() {
	for {
		r, isopen := <-p.reqChan
		if isopen {
			prevBlock := p.Get()
			r.RespChan <- rpcbus.NewResponse(prevBlock, nil)
		}
	}
}

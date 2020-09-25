package chain

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/sharedsafe"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// lastBlockProvider stores and provides blockchain tip of the local chain state
// Last Block will be the same with network tip only if the node is fully synced
type lastBlockProvider struct {
	lastBlock *sharedsafe.Object
	reqChan   <-chan rpcbus.Request
}

func newLastBlockProvider(rpcBus *rpcbus.RPCBus, b *block.Block) (*lastBlockProvider, error) {

	// set up rpcbus channels
	reqChan := make(chan rpcbus.Request, 1)
	if err := rpcBus.Register(topics.GetLastBlock, reqChan); err != nil {
		return nil, err
	}

	p := &lastBlockProvider{
		reqChan:   reqChan,
		lastBlock: new(sharedsafe.Object),
	}

	p.Set(b)

	go p.Listen()
	return p, nil
}

// Set updates last block with a full copy of the new block
func (p *lastBlockProvider) Set(n *block.Block) {
	p.lastBlock.Set(n)
}

// Get returns a safe copy of last block
func (p *lastBlockProvider) Get() block.Block {
	return p.lastBlock.Get().(block.Block)
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

package collection

import (
	"bytes"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
)

type BlockCollector struct {
	round uint64
	step  uint8
	queue *user.Queue
}

// ReceiveBestBlock will receive block candidate messages from the wire for a
// certain amount of time. If a block candidate message is received, who's hash
// is equal to the block hash that was passed into the function, this block
// is returned. If the time runs out before this happens, we return nil.
func (b *BlockCollector) ReceiveBestBlock(timerLength time.Duration,
	bestBlockHash []byte, blockChannel chan *payload.MsgCandidate,
	resultChannel chan *block.Block) {

	timer := time.NewTimer(timerLength)

	for {
		select {
		case <-timer.C:
			return
		case m := <-blockChannel:
			// If this is not the block that we received the hash for, discard it
			if !bytes.Equal(m.Block.Header.Hash, bestBlockHash) {
				break
			}

			resultChannel <- m.Block
			return
		}
	}
}

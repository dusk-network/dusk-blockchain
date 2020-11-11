package peer

import (
	"bytes"
	"net"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// ReaderFactory is responsible for spawning peers. It provides them with the
// reference to the message processor, which will process the received messages.
type ReaderFactory struct {
	processor *MessageProcessor
}

// NewReaderFactory returns an initialized ReaderFactory.
func NewReaderFactory(processor *MessageProcessor) *ReaderFactory {
	return &ReaderFactory{processor}
}

// SpawnReader returns a Reader. It will still need to be launched by
// running ReadLoop in a goroutine.
func (f *ReaderFactory) SpawnReader(conn net.Conn, gossip *processing.Gossip, dupeMap *dupemap.DupeMap, publisher eventbus.Publisher, rpcBus *rpcbus.RPCBus, responseChan chan<- *bytes.Buffer, exitChan chan<- struct{}) (*Reader, error) {
	pconn := &Connection{
		Conn:   conn,
		gossip: gossip,
	}

	reader := &Reader{
		Connection:   pconn,
		exitChan:     exitChan,
		responseChan: responseChan,
		processor:    f.processor,
	}

	// On each new connection the node sends topics.Mempool to retrieve mempool
	// txs from the new peer
	go func() {
		b := topics.MemPool.ToBuffer()
		responseChan <- &b
	}()

	return reader, nil
}

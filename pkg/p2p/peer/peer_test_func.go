package peer

import (
	"bytes"
	"net"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// StartPeerReader creates a Peer reader for testing purposes. The reader is
// at the receiving end of a binary-serialized Gossip notification and communicates through the
// eventbus
func StartPeerReader(conn net.Conn, bus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, counter *chainsync.Counter, responseChan chan<- *bytes.Buffer) *Reader {
	dupeMap := dupemap.NewDupeMap(5)
	exitChan := make(chan struct{}, 1)
	return NewReader(conn, processing.NewGossip(protocol.TestNet), dupeMap, bus, rpcBus, counter, responseChan, exitChan)
}

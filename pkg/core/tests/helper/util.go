package helper

import (
	"bytes"
	"crypto/rand"
	"net"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/stretchr/testify/assert"
)

// TxsToBuffer converts a slice of transactions to a bytes.Buffer.
func TxsToBuffer(t *testing.T, txs []transactions.Transaction) *bytes.Buffer {
	buf := new(bytes.Buffer)

	for _, tx := range txs {
		err := message.MarshalTx(buf, tx)
		if err != nil {
			assert.Nil(t, err)
		}
	}

	return bytes.NewBuffer(buf.Bytes())
}

// RandomSlice returns a random slice of size `size`
func RandomSlice(t *testing.T, size uint32) []byte {
	randSlice := make([]byte, size)
	_, err := rand.Read(randSlice)
	assert.Nil(t, err)
	return randSlice
}

// StartPeerReader creates a Peer reader for testing purposese. The reader is
// at the receiving end of a binary-serialized Gossip notification and communicates through the
// eventbus
func StartPeerReader(conn net.Conn, bus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, counter *chainsync.Counter, responseChan chan<- *bytes.Buffer) (*peer.Reader, error) {
	dupeMap := dupemap.NewDupeMap(5)
	exitChan := make(chan struct{}, 1)
	return peer.NewReader(conn, processing.NewGossip(protocol.TestNet), dupeMap, bus, rpcBus, counter, responseChan, exitChan)
}

package helper

import (
	"bytes"
	"crypto/rand"
	"net"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-wallet/v2/transactions"
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

func StartPeerReader(conn net.Conn) *peer.Reader {
	exitChan := make(chan struct{}, 1)
	r := peer.NewReader(conn, processing.NewGossip(protocol.TestNet), exitChan)
	return r
}

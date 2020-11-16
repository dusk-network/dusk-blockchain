package message_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeGetBlocks(t *testing.T) {
	var hashes [][]byte
	for i := 0; i < 5; i++ {
		hash, _ := crypto.RandEntropy(32)
		hashes = append(hashes, hash)
	}

	getBlocks := &message.GetBlocks{hashes}
	buf := new(bytes.Buffer)
	if err := getBlocks.Encode(buf); err != nil {
		t.Fatal(err)
	}

	getBlocks2 := &message.GetBlocks{}
	if err := getBlocks2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, getBlocks, getBlocks2)
}

package peermsg_test

import (
	"bytes"
	"testing"

	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeGetBlocks(t *testing.T) {
	var hashes [][]byte
	for i := 0; i < 5; i++ {
		hash, _ := crypto.RandEntropy(32)
		hashes = append(hashes, hash)
	}

	getBlocks := &peermsg.GetBlocks{hashes}
	buf := new(bytes.Buffer)
	if err := getBlocks.Encode(buf); err != nil {
		t.Fatal(err)
	}

	getBlocks2 := &peermsg.GetBlocks{}
	if err := getBlocks2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, getBlocks, getBlocks2)
}

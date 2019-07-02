package peermsg_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
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

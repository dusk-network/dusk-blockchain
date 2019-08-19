package peermsg_test

import (
	"bytes"
	"testing"

	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeInventory(t *testing.T) {
	hash, _ := crypto.RandEntropy(32)
	inv := &peermsg.Inv{}
	inv.AddItem(peermsg.InvTypeBlock, hash)
	buf := new(bytes.Buffer)
	if err := inv.Encode(buf); err != nil {
		t.Fatal(err)
	}

	inv2 := &peermsg.Inv{}
	if err := inv2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, inv, inv2)
}

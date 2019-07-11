package peermsg_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
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

package peermsg_test

import (
	"bytes"
	"math"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	crypto "github.com/dusk-network/dusk-crypto/hash"
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

func TestSizeLimit(t *testing.T) {
	// Encoding
	hash, _ := crypto.RandEntropy(32)
	inv := &peermsg.Inv{}
	for i := 0; uint32(i) < config.Get().Mempool.MaxInvItems+1; i++ {
		inv.AddItem(peermsg.InvTypeBlock, hash)
	}

	buf := new(bytes.Buffer)
	assert.Error(t, inv.Encode(buf))

	// Decoding
	// We encode an inv message manually, so we dont actually have
	// to create one with math.MaxUint64 items in it.
	buf = new(bytes.Buffer)
	assert.NoError(t, encoding.WriteVarInt(buf, math.MaxUint64))

	inv = &peermsg.Inv{}
	assert.Error(t, inv.Decode(buf))
}

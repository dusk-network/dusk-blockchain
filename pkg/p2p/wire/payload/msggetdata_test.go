package payload_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

func TestMsgGetDataEncodeDecodeTx(t *testing.T) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}
	sig, _ := crypto.RandEntropy(2000)

	txPubKey, _ := crypto.RandEntropy(32)
	pl := transactions.NewStandard(100)
	s := transactions.NewTX(transactions.StandardType, pl)
	in := transactions.NewInput(txPubKey, txPubKey, 0, sig)
	pl.AddInput(in)
	s.R = txPubKey

	out := transactions.NewOutput(200, byte32, sig)
	pl.AddOutput(out)
	if err := s.SetHash(); err != nil {
		t.Fatal(err)
	}

	msg := payload.NewMsgGetData()
	msg.AddTx(s)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := payload.NewMsgGetData()
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)
}

func TestMsgGetDataEncodeDecodeBlock(t *testing.T) {
	b := block.NewBlock()

	// Add 10 transactions
	for i := 0; i < 10; i++ {
		byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

		sig, _ := crypto.RandEntropy(2000)

		txPubKey, _ := crypto.RandEntropy(32)
		pl := transactions.NewStandard(100)
		s := transactions.NewTX(transactions.StandardType, pl)
		in := transactions.NewInput(txPubKey, txPubKey, 0, sig)
		pl.AddInput(in)
		s.R = txPubKey

		out := transactions.NewOutput(200, byte32, sig)
		pl.AddOutput(out)
		if err := s.SetHash(); err != nil {
			t.Fatal(err)
		}

		b.AddTx(s)
	}

	// Spoof previous hash and seed
	h, _ := crypto.RandEntropy(32)
	b.Header.PrevBlock = h

	s, _ := crypto.RandEntropy(33)
	b.Header.Seed = s

	// Add cert image
	rand1, _ := crypto.RandEntropy(32)
	rand2, _ := crypto.RandEntropy(32)

	sig, _ := crypto.RandEntropy(33)

	slice := make([][]byte, 0)
	slice = append(slice, rand1)
	slice = append(slice, rand2)

	cert := &block.Certificate{
		BRBatchedSig: sig,
		BRStep:       4,
		BRPubKeys:    slice,
		SRBatchedSig: sig,
		SRStep:       2,
		SRPubKeys:    slice,
	}

	if err := cert.SetHash(); err != nil {
		t.Fatal(err)
	}

	if err := b.AddCertHash(cert); err != nil {
		t.Fatal(err)
	}

	// Finish off
	if err := b.SetRoot(); err != nil {
		t.Fatal(err)
	}

	b.Header.Timestamp = time.Now().Unix()
	if err := b.SetHash(); err != nil {
		t.Fatal(err)
	}

	msg := payload.NewMsgGetData()
	msg.AddBlock(b)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := payload.NewMsgGetData()
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)
}

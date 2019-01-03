package payload

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

func TestMsgHeadersEncodeDecode(t *testing.T) {
	msg := NewMsgHeaders()

	// Add 10 block headers
	for i := 0; i < 10; i++ {
		block := NewBlock()

		// Add 10 transactions
		for i := 0; i < 10; i++ {
			byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

			sig, _ := crypto.RandEntropy(2000)

			txPubKey, _ := crypto.RandEntropy(32)
			s := transactions.NewTX()
			in := transactions.NewInput(txPubKey, txPubKey, 0, sig)
			s.AddInput(in)
			s.AddTxPubKey(txPubKey)

			out := transactions.NewOutput(200, byte32, sig)
			s.AddOutput(out)
			if err := s.SetHash(); err != nil {
				t.Fatal(err)
			}

			block.AddTx(s)
		}

		// Spoof previous hash and seed
		h, _ := crypto.RandEntropy(32)
		block.Header.PrevBlock = h
		block.Header.Seed = h

		// Add cert image
		rand1, _ := crypto.RandEntropy(32)
		rand2, _ := crypto.RandEntropy(32)

		sig, _ := crypto.RandEntropy(32)

		cert := NewCertificate(sig)
		for i := 1; i < 4; i++ {
			step := NewStep(uint32(i))
			step.AddData(rand1, rand2)
			cert.AddStep(step)
		}

		if err := cert.SetHash(); err != nil {
			t.Fatal(err)
		}

		if err := block.AddCertImage(cert); err != nil {
			t.Fatal(err)
		}

		// Finish off
		if err := block.SetRoot(); err != nil {
			t.Fatal(err)
		}

		block.SetTime(time.Now().Unix())
		if err := block.SetHash(); err != nil {
			t.Fatal(err)
		}

		msg.AddHeader(block.Header)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := NewMsgHeaders()
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)
}

package consensusmsg

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

func TestCandidateEncodeDecode(t *testing.T) {
	b := block.NewBlock()

	// Add 10 standard transactions
	for i := 0; i < 10; i++ {
		byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

		sig, _ := crypto.RandEntropy(2000)

		txPubKey, _ := crypto.RandEntropy(32)
		s := transactions.NewTX(transactions.StandardType, nil)
		in := transactions.NewInput(txPubKey, txPubKey, 0, sig)
		s.AddInput(in)
		s.AddTxPubKey(txPubKey)

		out := transactions.NewOutput(200, byte32, sig)
		s.AddOutput(out)
		if err := s.SetHash(); err != nil {
			t.Fatal(err)
		}

		b.AddTx(s)
	}

	// Spoof previous hash and seed
	h, _ := crypto.RandEntropy(32)
	b.Header.PrevBlock = h
	b.Header.Seed = h

	// Add cert image
	rand1, _ := crypto.RandEntropy(32)
	rand2, _ := crypto.RandEntropy(32)

	sig, _ := crypto.RandEntropy(32)

	cert := block.NewCertificate(sig)
	for i := 1; i < 4; i++ {
		step := block.NewStep(uint32(i))
		step.AddData(rand1, rand2)
		cert.AddStep(step)
	}

	if err := cert.SetHash(); err != nil {
		t.Fatal(err)
	}

	if err := b.AddCertImage(cert); err != nil {
		t.Fatal(err)
	}

	// Finish off
	if err := b.SetRoot(); err != nil {
		t.Fatal(err)
	}

	b.SetTime(time.Now().Unix())
	if err := b.SetHash(); err != nil {
		t.Fatal(err)
	}

	pl := NewCandidate(b)

	buf := new(bytes.Buffer)
	if err := pl.Encode(buf); err != nil {
		t.Fatal(err)
	}

	pl2 := &Candidate{}
	if err := pl2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, pl, pl2)
}

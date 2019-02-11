package consensusmsg

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestSigSetReductionEncodeDecode(t *testing.T) {
	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	blsSig, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := NewSigSetReduction(byte32, byte32, blsSig, byte32)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &SigSetReduction{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}

// Check to see whether length checks are working.
func TestSigSetReductionChecks(t *testing.T) {
	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	blsSig, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewSigSetReduction(blsSig, byte32, blsSig, byte32); err == nil {
		t.Fatal("check for winningblock did not work")
	}

	if _, err := NewSigSetReduction(byte32, blsSig, blsSig, byte32); err == nil {
		t.Fatal("check for sigsethash did not work")
	}

	if _, err := NewSigSetReduction(byte32, byte32, byte32, byte32); err == nil {
		t.Fatal("check for sigbls did not work")
	}
}

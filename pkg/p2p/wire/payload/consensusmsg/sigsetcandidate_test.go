package consensusmsg

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestSigSetCandidateEncodeDecode(t *testing.T) {
	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	sigs, err := crypto.RandEntropy(200)
	if err != nil {
		t.Fatal(err)
	}

	sigBLS, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := NewSigSetCandidate(byte32, sigs, byte32, sigBLS)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &SigSetCandidate{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}

// Check to see whether length checks are working.
func TestSigSetCandidateChecks(t *testing.T) {
	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	sigs, err := crypto.RandEntropy(200)
	if err != nil {
		t.Fatal(err)
	}

	wrongByte32, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewSigSetCandidate(wrongByte32, sigs, byte32, wrongByte32); err == nil {
		t.Fatal("check for winningblock did not work")
	}

	if _, err := NewSigSetCandidate(byte32, sigs, byte32, byte32); err == nil {
		t.Fatal("check for winningblock did not work")
	}
}

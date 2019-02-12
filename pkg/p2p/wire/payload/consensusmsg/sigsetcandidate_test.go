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

	pkBLS, err := crypto.RandEntropy(129)
	if err != nil {
		t.Fatal(err)
	}

	sigBLS, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	var votes []*Vote
	for i := 0; i < 5; i++ {
		vote, err := NewVote(byte32, pkBLS, sigBLS, 1)
		if err != nil {
			t.Fatal(err)
		}

		votes = append(votes, vote)
	}

	msg, err := NewSigSetCandidate(byte32, votes, byte32)
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

	pkBLS, err := crypto.RandEntropy(129)
	if err != nil {
		t.Fatal(err)
	}

	wrongByte32, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	var votes []*Vote
	for i := 0; i < 5; i++ {
		vote, err := NewVote(byte32, pkBLS, wrongByte32, 1)
		if err != nil {
			t.Fatal(err)
		}

		votes = append(votes, vote)
	}

	if _, err := NewSigSetCandidate(wrongByte32, votes, byte32); err == nil {
		t.Fatal("check for winningblock did not work")
	}
}

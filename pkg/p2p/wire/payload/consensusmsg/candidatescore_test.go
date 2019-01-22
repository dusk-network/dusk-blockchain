package consensusmsg

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestCandidateScoreEncodeDecode(t *testing.T) {
	proof, err := crypto.RandEntropy(200)
	if err != nil {
		t.Fatal(err)
	}

	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := NewCandidateScore(200, proof, byte32, byte32)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &CandidateScore{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}

// Check to see whether length checks are working.
func TestCandidateScoreChecks(t *testing.T) {
	proof, err := crypto.RandEntropy(200)
	if err != nil {
		t.Fatal(err)
	}

	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	wrongByte32, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewCandidateScore(200, proof, wrongByte32, byte32); err == nil {
		t.Fatal("check for candidatehash did not work")
	}

	if _, err := NewCandidateScore(200, proof, byte32, wrongByte32); err == nil {
		t.Fatal("check for seed did not work")
	}
}

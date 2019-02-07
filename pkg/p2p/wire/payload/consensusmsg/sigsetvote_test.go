package consensusmsg

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestSigSetVoteEncodeDecode(t *testing.T) {
	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	blsSig, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := NewSigSetVote(byte32, byte32, blsSig, byte32, blsSig)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &SigSetVote{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}

// Check to see whether length checks are working.
func TestSigSetVoteChecks(t *testing.T) {
	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	blsSig, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewSigSetVote(blsSig, byte32, blsSig, byte32, blsSig); err == nil {
		t.Fatal("check for winningblock did not work")
	}

	if _, err := NewSigSetVote(byte32, blsSig, blsSig, byte32, blsSig); err == nil {
		t.Fatal("check for sigsethash did not work")
	}

	if _, err := NewSigSetVote(byte32, byte32, byte32, byte32, blsSig); err == nil {
		t.Fatal("check for sigbls did not work")
	}

	if _, err := NewSigSetVote(byte32, byte32, blsSig, byte32, byte32); err == nil {
		t.Fatal("check for score did not work")
	}
}

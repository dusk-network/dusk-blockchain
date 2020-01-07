package candidate

import (
	"bytes"
	"testing"
)

// Ensure that the behaviour of the validator works as intended.
// It should republish blocks with a correct hash and root.
func TestValidatorValidBlock(t *testing.T) {
	cm := mockCandidateMessage(t)
	buf := new(bytes.Buffer)
	if err := Encode(buf, cm); err != nil {
		t.Fatal(err)
	}

	// Send it over to the validator
	if err := Validate(*buf); err != nil {
		t.Fatal(err)
	}
}

// Ensure that blocks with an invalid hash or tx root will not be
// republished.
func TestValidatorInvalidBlock(t *testing.T) {
	cm := mockCandidateMessage(t)
	buf := new(bytes.Buffer)
	if err := Encode(buf, cm); err != nil {
		t.Fatal(err)
	}

	// Remove one of the transactions to remove the integrity of
	// the merkle root
	cm.Block.Txs = cm.Block.Txs[1:]
	buf = new(bytes.Buffer)
	if err := Encode(buf, cm); err != nil {
		t.Fatal(err)
	}

	if err := Validate(*buf); err == nil {
		t.Fatal("processing a block with an invalid hash should return an error")
	}
}

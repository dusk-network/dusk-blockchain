package msg_test

import (
	"bytes"
	"testing"

	"github.com/magiconair/properties/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestVoteEncodeDecode(t *testing.T) {
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

	vote, err := NewVote(byte32, pkBLS, sigBLS, 1)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := vote.Encode(buf); err != nil {
		t.Fatal(err)
	}

	vote2 := &Vote{}
	if err := vote2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, vote, vote2)
}

func TestVoteChecks(t *testing.T) {
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

	if _, err := NewVote(sigBLS, pkBLS, sigBLS, 1); err == nil {
		t.Fatal("check for hash did not work")
	}

	if _, err := NewVote(byte32, pkBLS, byte32, 1); err == nil {
		t.Fatal("check for sig did not work")
	}
}

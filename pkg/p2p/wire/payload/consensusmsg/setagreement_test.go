package consensusmsg

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestSetAgreementEncodeDecode(t *testing.T) {
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

	msg, err := NewSetAgreement(byte32, votes)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &SetAgreement{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}

func TestSetAgreementChecks(t *testing.T) {
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

	if _, err := NewSetAgreement(sigBLS, votes); err == nil {
		t.Fatal("check for hash did not work")
	}
}

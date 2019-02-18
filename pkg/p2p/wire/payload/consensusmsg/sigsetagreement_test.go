package consensusmsg

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestSigSetAgreementEncodeDecode(t *testing.T) {
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

	msg, err := NewSigSetAgreement(byte32, byte32, votes, 1)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &SigSetAgreement{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}

func TestSigSetAgreementChecks(t *testing.T) {
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

	if _, err := NewSigSetAgreement(sigBLS, byte32, votes, 1); err == nil {
		t.Fatal("check for blockhash did not work")
	}

	if _, err := NewSigSetAgreement(byte32, sigBLS, votes, 1); err == nil {
		t.Fatal("check for sethash did not work")
	}
}

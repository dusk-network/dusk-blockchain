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

	voteSet, err := crypto.RandEntropy(300)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := NewSetAgreement(byte32, voteSet)
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
	wrongByte32, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	voteSet, err := crypto.RandEntropy(300)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewSetAgreement(wrongByte32, voteSet); err == nil {
		t.Fatal("check for hash did not work")
	}
}

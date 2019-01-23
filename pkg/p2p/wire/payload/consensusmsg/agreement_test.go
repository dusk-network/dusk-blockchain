package consensusmsg

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestAgreementEncodeDecode(t *testing.T) {
	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	blsSig, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := NewAgreement(blsSig, true, 4, byte32, blsSig, byte32)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &Agreement{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}

// Check to see whether length checks are working.
func TestAgreementChecks(t *testing.T) {
	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	wrongByte32, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewAgreement(byte32, true, 4, byte32, wrongByte32, byte32); err == nil {
		t.Fatal("check for score did not work")
	}

	if _, err := NewAgreement(byte32, true, 4, wrongByte32, wrongByte32, byte32); err == nil {
		t.Fatal("check for hash did not work")
	}

	if _, err := NewAgreement(wrongByte32, true, 4, byte32, byte32, byte32); err == nil {
		t.Fatal("check for sigbls did not work")
	}
}

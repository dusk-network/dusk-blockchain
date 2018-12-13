package payload

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/crypto"
)

func TestMsgBinaryEncodeDecode(t *testing.T) {
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	sigBLS, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	sigEd, err := crypto.RandEntropy(64)
	if err != nil {
		t.Fatal(err)
	}

	pubKey, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := NewMsgBinary(200, true, hash, sigBLS, sigEd, pubKey)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &MsgBinary{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}

// Check to see whether length checks are working.
func TestMsgBinaryChecks(t *testing.T) {
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	wrongHash, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	sigBLS, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	wrongSigBLS, err := crypto.RandEntropy(35)
	if err != nil {
		t.Fatal(err)
	}

	sigEd, err := crypto.RandEntropy(64)
	if err != nil {
		t.Fatal(err)
	}

	wrongSigEd, err := crypto.RandEntropy(58)
	if err != nil {
		t.Fatal(err)
	}

	pubKey, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	wrongPubKey, err := crypto.RandEntropy(30)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewMsgBinary(200, true, wrongHash, sigBLS, sigEd, pubKey); err == nil {
		t.Fatal("check for hash did not work")
	}

	if _, err := NewMsgBinary(200, true, hash, wrongSigBLS, sigEd, pubKey); err == nil {
		t.Fatal("check for sig did not work")
	}

	if _, err := NewMsgBinary(200, true, hash, sigBLS, wrongSigEd, pubKey); err == nil {
		t.Fatal("check for pubkey did not work")
	}

	if _, err := NewMsgBinary(200, true, hash, sigBLS, sigEd, wrongPubKey); err == nil {
		t.Fatal("check for pubkey did not work")
	}
}

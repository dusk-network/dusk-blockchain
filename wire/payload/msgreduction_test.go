package payload

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/crypto"
)

func TestMsgReductionEncodeDecode(t *testing.T) {
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	sig, err := crypto.RandEntropy(64)
	if err != nil {
		t.Fatal(err)
	}

	pubKey, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := NewMsgReduction(200, hash, sig, pubKey)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &MsgReduction{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}

// Check to see whether length checks are working.
func TestMsgReductionChecks(t *testing.T) {
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	wrongHash, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	sig, err := crypto.RandEntropy(64)
	if err != nil {
		t.Fatal(err)
	}

	wrongSig, err := crypto.RandEntropy(62)
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

	if _, err := NewMsgReduction(200, wrongHash, sig, pubKey); err == nil {
		t.Fatal("check for hash did not work")
	}

	if _, err := NewMsgReduction(200, hash, wrongSig, pubKey); err == nil {
		t.Fatal("check for sig did not work")
	}

	if _, err := NewMsgReduction(200, hash, sig, wrongPubKey); err == nil {
		t.Fatal("check for pubkey did not work")
	}
}

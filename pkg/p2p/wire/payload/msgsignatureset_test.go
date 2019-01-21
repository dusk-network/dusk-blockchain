package payload

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestMsgSignatureSetEncodeDecode(t *testing.T) {
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	sigs, err := crypto.RandEntropy(200)
	if err != nil {
		t.Fatal(err)
	}

	sigEd, err := crypto.RandEntropy(64)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := NewMsgSignatureSet(300, 15000, hash, hash, sigs, sigEd, hash)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &MsgSignatureSet{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}

// Check to see whether length checks are working.
func TestMsgSignatureSetChecks(t *testing.T) {
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	sigs, err := crypto.RandEntropy(200)
	if err != nil {
		t.Fatal(err)
	}

	sigEd, err := crypto.RandEntropy(64)
	if err != nil {
		t.Fatal(err)
	}

	wrongHash, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	wrongSig, err := crypto.RandEntropy(62)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewMsgSignatureSet(300, 15000, wrongHash, hash, sigs, sigEd, hash); err == nil {
		t.Fatal("check for prevblock did not work")
	}

	if _, err := NewMsgSignatureSet(300, 15000, hash, wrongHash, sigs, sigEd, hash); err == nil {
		t.Fatal("check for winningblock did not work")
	}

	if _, err := NewMsgSignatureSet(300, 15000, hash, hash, sigs, wrongSig, hash); err == nil {
		t.Fatal("check for sig did not work")
	}

	if _, err := NewMsgSignatureSet(300, 15000, hash, hash, sigs, sigEd, wrongHash); err == nil {
		t.Fatal("check for pk did not work")
	}
}

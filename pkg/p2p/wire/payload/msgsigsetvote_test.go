package payload

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestMsgSigSetVoteEncodeDecode(t *testing.T) {
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

	msg, err := NewMsgSigSetVote(300, 15000, 2, hash, sigs, hash, sigEd, hash, hash)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &MsgSigSetVote{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}

// Check to see whether length checks are working.
func TestMsgSigSetVoteChecks(t *testing.T) {
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

	if _, err := NewMsgSigSetVote(300, 15000, 2, wrongHash, sigs, hash, sigEd, hash, hash); err == nil {
		t.Fatal("check for prevblock did not work")
	}

	if _, err := NewMsgSigSetVote(300, 15000, 2, hash, sigs, wrongHash, sigEd, hash, hash); err == nil {
		t.Fatal("check for sigbls did not work")
	}

	if _, err := NewMsgSigSetVote(300, 15000, 2, hash, sigs, hash, wrongSig, hash, hash); err == nil {
		t.Fatal("check for siged did not work")
	}

	if _, err := NewMsgSigSetVote(300, 15000, 2, hash, sigs, hash, sigEd, wrongHash, hash); err == nil {
		t.Fatal("check for pkbls did not work")
	}

	if _, err := NewMsgSigSetVote(300, 15000, 2, hash, sigs, hash, sigEd, hash, wrongHash); err == nil {
		t.Fatal("check for pked did not work")
	}
}

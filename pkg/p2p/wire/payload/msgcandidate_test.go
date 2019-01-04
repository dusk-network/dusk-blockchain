package payload

import (
	"bytes"
	"crypto/rand"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestMsgCandidateEncodeDecode(t *testing.T) {
	pk, sk, err := bls.GenKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	sig, err := bls.Sign(sk, hash)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := NewMsgCandidate(hash, sig, pk)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &MsgCandidate{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}

// Check to see whether length checks are working.
func TestMsgCandidateChecks(t *testing.T) {
	pk, sk, err := bls.GenKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	sig, err := bls.Sign(sk, hash)
	if err != nil {
		t.Fatal(err)
	}

	wrongHash, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewMsgCandidate(wrongHash, sig, pk); err == nil {
		t.Fatal("check for hash did not work")
	}
}

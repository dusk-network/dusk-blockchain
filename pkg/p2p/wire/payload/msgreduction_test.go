package payload

import (
	"bytes"
	"crypto/rand"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"golang.org/x/crypto/ed25519"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestMsgReductionEncodeDecode(t *testing.T) {
	pk, sk, err := bls.GenKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	edpk, edsk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	sigBLS, err := bls.Sign(sk, hash)
	if err != nil {
		t.Fatal(err)
	}

	sigEd := ed25519.Sign(edsk, hash)

	msg, err := NewMsgReduction(sigBLS, hash, hash, sigEd, &edpk, sigBLS, pk, 200, 23000, 1)
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
	pk, sk, err := bls.GenKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	edpk, edsk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	sigBLS, err := bls.Sign(sk, hash)
	if err != nil {
		t.Fatal(err)
	}

	sigEd := ed25519.Sign(edsk, hash)

	wrongHash, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	wrongSig, err := crypto.RandEntropy(62)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewMsgReduction(sigBLS, wrongHash, hash, sigEd, &edpk, sigBLS, pk, 200, 23000, 1); err == nil {
		t.Fatal("check for hash did not work")
	}

	if _, err := NewMsgReduction(sigBLS, hash, wrongHash, sigEd, &edpk, sigBLS, pk, 200, 23000, 1); err == nil {
		t.Fatal("check for prevhash did not work")
	}

	if _, err := NewMsgReduction(sigBLS, hash, hash, wrongSig, &edpk, sigBLS, pk, 200, 23000, 1); err == nil {
		t.Fatal("check for siged did not work")
	}
}

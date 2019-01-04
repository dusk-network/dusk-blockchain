package payload

import (
	"bytes"
	"crypto/rand"
	"testing"

	"golang.org/x/crypto/ed25519"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestMsgBinaryEncodeDecode(t *testing.T) {
	pk, sk, err := bls.GenKeyPair(rand.Reader)
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

	edpk, edsk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	sigEd := ed25519.Sign(edsk, hash)

	msg, err := NewMsgBinary(sigBLS, true, hash, hash, sigEd, &edpk, sigBLS, pk, 200, 230000, 4)
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
	pk, sk, err := bls.GenKeyPair(rand.Reader)
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

	edpk, edsk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	sigEd := ed25519.Sign(edsk, hash)

	wrongHash, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	wrongSigEd, err := crypto.RandEntropy(58)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewMsgBinary(sigBLS, true, wrongHash, hash, sigEd, &edpk, sigBLS, pk, 200, 230000, 4); err == nil {
		t.Fatal("check for hash did not work")
	}

	if _, err := NewMsgBinary(sigBLS, true, hash, wrongHash, sigEd, &edpk, sigBLS, pk, 200, 230000, 4); err == nil {
		t.Fatal("check for prevhash did not work")
	}

	if _, err := NewMsgBinary(sigBLS, true, hash, hash, wrongSigEd, &edpk, sigBLS, pk, 200, 230000, 4); err == nil {
		t.Fatal("check for siged did not work")
	}
}

package payload_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
)

func TestMsgCertificateEncodeDecode(t *testing.T) {
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	rand1, _ := crypto.RandEntropy(32)
	rand2, _ := crypto.RandEntropy(32)

	sig, _ := crypto.RandEntropy(33)

	slice := make([][]byte, 0)
	slice = append(slice, rand1)
	slice = append(slice, rand2)

	cert := &block.Certificate{
		BRBatchedSig:      sig,
		BRStep:            4,
		BRPubKeys:         slice,
		BRSortitionProofs: slice,
		SRBatchedSig:      sig,
		SRStep:            2,
		SRPubKeys:         slice,
		SRSortitionProofs: slice,
	}

	if err := cert.SetHash(); err != nil {
		t.Fatal(err)
	}

	msg, err := payload.NewMsgCertificate(1500, hash, cert)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &payload.MsgCertificate{}
	msg2.Decode(buf)

	assert.Equal(t, msg, msg2)
}

// Check to see whether length checks are working.
func TestMsgCertificateChecks(t *testing.T) {
	wrongHash, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	rand1, _ := crypto.RandEntropy(32)
	rand2, _ := crypto.RandEntropy(32)

	sig, _ := crypto.RandEntropy(33)

	slice := make([][]byte, 0)
	slice = append(slice, rand1)
	slice = append(slice, rand2)

	cert := &block.Certificate{
		BRBatchedSig:      sig,
		BRStep:            4,
		BRPubKeys:         slice,
		BRSortitionProofs: slice,
		SRBatchedSig:      sig,
		SRStep:            2,
		SRPubKeys:         slice,
		SRSortitionProofs: slice,
	}

	if _, err := payload.NewMsgCertificate(200, wrongHash, cert); err == nil {
		t.Fatal("check for hash did not work")
	}
}

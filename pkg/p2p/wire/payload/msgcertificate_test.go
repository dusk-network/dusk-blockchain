package payload

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestMsgCertificateEncodeDecode(t *testing.T) {
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	rand1, _ := crypto.RandEntropy(32)
	rand2, _ := crypto.RandEntropy(32)

	sig, _ := crypto.RandEntropy(32)

	cert := NewCertificate(sig)
	for i := 1; i < 4; i++ {
		step := NewStep(uint32(i))
		step.AddData(rand1, rand2)
		cert.AddStep(step)
	}

	if err := cert.SetHash(); err != nil {
		t.Fatal(err)
	}

	msg, err := NewMsgCertificate(1500, hash, cert)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &MsgCertificate{}
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

	sig, _ := crypto.RandEntropy(32)

	cert := NewCertificate(sig)
	for i := 1; i < 4; i++ {
		step := NewStep(uint32(i))
		step.AddData(rand1, rand2)
		cert.AddStep(step)
	}

	if _, err := NewMsgCertificate(200, wrongHash, cert); err == nil {
		t.Fatal("check for hash did not work")
	}
}

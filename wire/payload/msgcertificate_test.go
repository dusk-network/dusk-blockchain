package payload

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/crypto"
)

func TestMsgCertificateEncodeDecode(t *testing.T) {
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	cert, err := crypto.RandEntropy(100)
	if err != nil {
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

	cert, err := crypto.RandEntropy(100)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewMsgCertificate(200, wrongHash, cert); err == nil {
		t.Fatal("check for hash did not work")
	}
}

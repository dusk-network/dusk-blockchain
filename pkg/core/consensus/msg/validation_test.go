package msg

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/key"
	"github.com/dusk-network/dusk-crypto/bls"
	"github.com/stretchr/testify/assert"
)

func TestVerifyBLSSignature(t *testing.T) {
	k, _ := key.NewRandConsensusKeys()
	payload := bytes.NewBufferString("this is a test message").Bytes()

	sig, _ := bls.Sign(k.BLSSecretKey, k.BLSPubKey, payload)
	s := sig.Compress()
	orig := make([]byte, 33)
	copy(orig, s)
	if !assert.NoError(t, VerifyBLSSignature(k.BLSPubKeyBytes, payload, s)) {
		t.FailNow()
	}

	assert.Equal(t, s, orig)
}

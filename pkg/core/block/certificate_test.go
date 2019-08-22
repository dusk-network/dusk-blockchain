package block_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeCert(t *testing.T) {
	assert := assert.New(t)

	// random certificate
	cert := helper.RandomCertificate(t)

	// Encode certificate into a buffer
	buf := new(bytes.Buffer)
	err := block.MarshalCertificate(buf, cert)
	assert.Nil(err)

	// Decode buffer into a certificate struct
	decCert := &block.Certificate{}
	err = block.UnmarshalCertificate(buf, decCert)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(cert.Equals(decCert))

}

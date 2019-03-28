package block_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
)

func TestEncodeDecodeCert(t *testing.T) {
	assert := assert.New(t)

	// random certificate
	cert := helper.RandomCertificate(t)

	// Encode certificate into a buffer
	buf := new(bytes.Buffer)
	err := cert.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a certificate struct
	decCert := &block.Certificate{}
	err = decCert.Decode(buf)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(cert.Equals(decCert))

}

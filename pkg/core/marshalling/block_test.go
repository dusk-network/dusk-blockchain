package marshalling_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeBlock(t *testing.T) {

	assert := assert.New(t)

	// random block
	blk := helper.RandomBlock(t, 200, 2)

	// Encode block into a buffer
	buf := new(bytes.Buffer)
	err := marshalling.MarshalBlock(buf, blk)
	assert.Nil(err)

	// Decode buffer into a block struct
	decBlk := block.NewBlock()
	err = marshalling.UnmarshalBlock(buf, decBlk)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(blk.Equals(decBlk))
}

func TestEncodeDecodeCert(t *testing.T) {
	assert := assert.New(t)

	// random certificate
	cert := helper.RandomCertificate(t)

	// Encode certificate into a buffer
	buf := new(bytes.Buffer)
	err := marshalling.MarshalCertificate(buf, cert)
	assert.Nil(err)

	// Decode buffer into a certificate struct
	decCert := &block.Certificate{}
	err = marshalling.UnmarshalCertificate(buf, decCert)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(cert.Equals(decCert))

}

func TestEncodeDecodeHeader(t *testing.T) {

	assert := assert.New(t)

	// Create a random header
	hdr := helper.RandomHeader(t, 200)
	err := hdr.SetHash()
	assert.Nil(err)

	// Encode header into a buffer
	buf := new(bytes.Buffer)
	err = marshalling.MarshalHeader(buf, hdr)
	assert.Nil(err)

	// Decode buffer into a header struct
	decHdr := block.NewHeader()
	err = marshalling.UnmarshalHeader(buf, decHdr)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(hdr.Equals(decHdr))
}

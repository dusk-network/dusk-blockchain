package block

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeHeader(t *testing.T) {

	assert := assert.New(t)

	// Create a random header
	hdr := randomHeader(t)
	err := hdr.SetHash()
	assert.Nil(err)

	// Encode header into a buffer
	buf := new(bytes.Buffer)
	err = hdr.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a header struct
	decHdr := &Header{}
	err = decHdr.Decode(buf)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(hdr.Equals(decHdr))
}

func randomHeader(t *testing.T) *Header {

	h := &Header{
		Version:   0,
		Timestamp: 2000,
		Height:    200,

		PrevBlock: randomSlice(t, 32),
		Seed:      randomSlice(t, 33),
		TxRoot:    randomSlice(t, 32),

		CertHash: randomSlice(t, 32),
	}

	return h
}

func randomSlice(t *testing.T, size uint32) []byte {
	randSlice := make([]byte, size)
	_, err := rand.Read(randSlice)
	assert.Nil(t, err)
	return randSlice
}

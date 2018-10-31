package encoding

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Simple test case for WriteVarBytes and ReadVarBytes. This test case won't do much,
// as the data is already represented in bytes. However, this will tell us if the functions
// work properly.
func TestVarBytesEncodeDecode(t *testing.T) {
	// Get a random number of bytes
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(2048)
	bs := randBytes(n)

	// Serialize
	buf := new(bytes.Buffer)
	if err := WriteVarBytes(buf, bs); err != nil {
		t.Fatalf("%v", err)
	}

	// Deserialize
	rbs, err := ReadVarBytes(buf)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Compare
	assert.Equal(t, bs, rbs)
}

// Simple test case for writing and reading strings.
func TestVarStringEncodeDecode(t *testing.T) {
	// Get a random string
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(2048)
	str := string(randBytes(n))

	// Serialize
	buf := new(bytes.Buffer)
	if err := WriteString(buf, str); err != nil {
		t.Fatalf("%v", err)
	}

	// Deserialize
	rstr, err := ReadString(buf)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Compare
	assert.Equal(t, str, rstr)
}

//https://stackoverflow.com/a/31832326/5203311
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}

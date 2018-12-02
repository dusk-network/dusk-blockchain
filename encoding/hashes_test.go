package encoding

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.dusk.network/dusk-core/dusk-go/crypto"
)

// Basic test. This won't do much as it's already in byte representation, but is more intended to show
// that the function works as intended for when it's writing to a buffer with other elements.
func TestHashEncodeDecode(t *testing.T) {
	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Serialize
	buf := new(bytes.Buffer)
	if err := WriteHash(buf, byte32); err != nil {
		t.Fatalf("%v", err)
	}

	// Check if it serialized correctly
	assert.Equal(t, buf.Bytes(), byte32)
	assert.Equal(t, len(buf.Bytes()), 32)

	// Deserialize
	hash, err := ReadHash(buf)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Content should be the same
	assert.Equal(t, hash, byte32)
}

// Test to make sure it takes only byte slices of length 32
func TestHashLength(t *testing.T) {
	byte16, err := crypto.RandEntropy(16)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Serialize
	buf := new(bytes.Buffer)
	err = WriteHash(buf, byte16) // This should fail
	if err == nil {
		t.Fatal("did not throw error when serializing byte slice of improper length")
	}
	fmt.Println(err)

	buf.Reset()
	byte40, err := crypto.RandEntropy(40)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Serialize
	err = WriteHash(buf, byte40) // This should also fail
	if err == nil {
		t.Fatal("did not throw error when serializing byte slice of improper length")
	}
	fmt.Println(err)
}

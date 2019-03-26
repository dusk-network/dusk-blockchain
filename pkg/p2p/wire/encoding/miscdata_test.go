package encoding

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

// Test bool functions
func TestBoolEncodeDecode(t *testing.T) {
	b := true
	buf := new(bytes.Buffer)
	if err := WriteBool(buf, b); err != nil {
		t.Fatal(err)
	}

	var c bool
	if err := ReadBool(buf, &c); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, b, c)
}

// Basic test. This won't do much as it's already in byte representation, but is more intended to show
// that the function works as intended for when it's writing to a buffer with other elements.
func Test256EncodeDecode(t *testing.T) {
	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize
	buf := new(bytes.Buffer)
	if err := Write256(buf, byte32); err != nil {
		t.Fatal(err)
	}

	// Check if it serialized correctly
	assert.Equal(t, buf.Bytes(), byte32)
	assert.Equal(t, len(buf.Bytes()), 32)

	// Deserialize
	var hash []byte
	if err := Read256(buf, &hash); err != nil {
		t.Fatal(err)
	}

	// Content should be the same
	assert.Equal(t, hash, byte32)
}

func Test512EncodeDecode(t *testing.T) {
	byte64, err := crypto.RandEntropy(64)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize
	buf := new(bytes.Buffer)
	if err := Write512(buf, byte64); err != nil {
		t.Fatal(err)
	}

	// Check if it serialized correctly
	assert.Equal(t, buf.Bytes(), byte64)
	assert.Equal(t, len(buf.Bytes()), 64)

	// Deserialize
	var hash []byte
	if err := Read512(buf, &hash); err != nil {
		t.Fatal(err)
	}

	// Content should be the same
	assert.Equal(t, hash, byte64)
}

// Test to make sure it only takes byte slices of length 32
func Test256Length(t *testing.T) {
	byte16, err := crypto.RandEntropy(16)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize
	buf := new(bytes.Buffer)
	err = Write256(buf, byte16) // This should fail
	if err == nil {
		t.Fatal("did not throw error when serializing byte slice of improper length")
	}

	buf.Reset()
	byte40, err := crypto.RandEntropy(40)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize
	err = Write256(buf, byte40) // This should also fail
	if err == nil {
		t.Fatal("did not throw error when serializing byte slice of improper length")
	}
}

// Test to make sure it only takes byte slices of length 64
func Test512Length(t *testing.T) {
	byte16, err := crypto.RandEntropy(16)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize
	buf := new(bytes.Buffer)
	err = Write512(buf, byte16) // This should fail
	if err == nil {
		t.Fatal("did not throw error when serializing byte slice of improper length")
	}

	buf.Reset()
	byte80, err := crypto.RandEntropy(80)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize
	err = Write512(buf, byte80) // This should also fail
	if err == nil {
		t.Fatal("did not throw error when serializing byte slice of improper length")
	}
}

func TestBLSEncodeDecode(t *testing.T) {
	byte33, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize
	buf := new(bytes.Buffer)
	if err := WriteBLS(buf, byte33); err != nil {
		t.Fatal(err)
	}

	// Check if it serialized correctly
	assert.Equal(t, buf.Bytes(), byte33)
	assert.Equal(t, len(buf.Bytes()), 33)

	// Deserialize
	var hash []byte
	if err := ReadBLS(buf, &hash); err != nil {
		t.Fatal(err)
	}

	// Content should be the same
	assert.Equal(t, hash, byte33)
}

func TestBLSLength(t *testing.T) {
	byte16, err := crypto.RandEntropy(16)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize
	buf := new(bytes.Buffer)
	err = WriteBLS(buf, byte16) // This should fail
	if err == nil {
		t.Fatal("did not throw error when serializing byte slice of improper length")
	}

	buf.Reset()
	byte80, err := crypto.RandEntropy(80)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize
	err = WriteBLS(buf, byte80) // This should also fail
	if err == nil {
		t.Fatal("did not throw error when serializing byte slice of improper length")
	}
}

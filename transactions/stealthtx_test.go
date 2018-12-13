package transactions

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/crypto"
)

// Test if GetEncodeSize returns the right size for the buffer
func TestGetEncodeSize(t *testing.T) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	// Input
	sig, _ := crypto.RandEntropy(2000)
	in := Input{
		byte32,
		byte32,
		1,
		sig,
	}

	// Output
	out := Output{
		200,
		byte32,
	}

	// Type attribute
	ta := TypeAttributes{
		[]Input{in},
		byte32,
		[]Output{out},
	}

	R, _ := crypto.RandEntropy(32)
	s := Stealth{
		1,
		1,
		R,
		ta,
	}

	// Get pre-determined buffer size
	size := s.GetEncodeSize()

	// Now encode using a standard buffer
	dynBuf := new(bytes.Buffer)
	if err := s.Encode(dynBuf); err != nil {
		t.Fatal(err)
	}

	// Compare size
	assert.Equal(t, uint64(len(dynBuf.Bytes())), size)
}

// Testing encoding/decoding functions for StealthTX
func TestEncodeDecode(t *testing.T) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	// Input
	sig, _ := crypto.RandEntropy(2000)
	in := Input{
		byte32,
		byte32,
		1,
		sig,
	}

	// Output
	out := Output{
		200,
		byte32,
	}

	// Type attribute
	ta := TypeAttributes{
		[]Input{in},
		byte32,
		[]Output{out},
	}

	R, _ := crypto.RandEntropy(32)
	s := Stealth{
		1,
		1,
		R,
		ta,
	}

	// Serialize
	size := s.GetEncodeSize()
	bs := make([]byte, 0, size)
	buf := bytes.NewBuffer(bs)

	if err := s.Encode(buf); err != nil {
		t.Fatal(err)
	}

	// Deserialize
	var newStealth Stealth

	if err := newStealth.Decode(buf); err != nil {
		t.Fatal(err)
	}

	// Compare
	assert.Equal(t, s, newStealth)
}

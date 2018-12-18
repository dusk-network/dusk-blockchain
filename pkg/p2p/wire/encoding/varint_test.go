package encoding

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompactSize(t *testing.T) {
	a := uint64(1)
	b := uint64(2 ^ 8)
	c := uint64(2 ^ 16)
	d := uint64(2 ^ 32)

	// Serialize
	buf := new(bytes.Buffer)
	if err := WriteVarInt(buf, uint64(a)); err != nil {
		t.Fatal(err)
	}
	if err := WriteVarInt(buf, uint64(b)); err != nil {
		t.Fatal(err)
	}
	if err := WriteVarInt(buf, uint64(c)); err != nil {
		t.Fatal(err)
	}
	if err := WriteVarInt(buf, uint64(d)); err != nil {
		t.Fatal(err)
	}

	// Deserialize
	e, err := ReadVarInt(buf)
	if err != nil {
		t.Fatal(err)
	}
	f, err := ReadVarInt(buf)
	if err != nil {
		t.Fatal(err)
	}
	g, err := ReadVarInt(buf)
	if err != nil {
		t.Fatal(err)
	}
	h, err := ReadVarInt(buf)
	if err != nil {
		t.Fatal(err)
	}

	// Compare
	assert.Equal(t, a, e)
	assert.Equal(t, b, f)
	assert.Equal(t, c, g)
	assert.Equal(t, d, h)
}

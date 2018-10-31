package encoding

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Default serialization test
func TestIntegerEncodeDecode(t *testing.T) {
	a := uint8(5)
	b := uint16(10)
	c := uint32(20)
	d := uint64(40)

	// Serialize
	buf := new(bytes.Buffer)

	if err := PutUint8(buf, a); err != nil {
		t.Fatalf("%v", err)
	}
	if err := PutUint16(buf, binary.LittleEndian, b); err != nil {
		t.Fatalf("%v", err)
	}
	if err := PutUint32(buf, binary.LittleEndian, c); err != nil {
		t.Fatalf("%v", err)
	}
	if err := PutUint64(buf, binary.LittleEndian, d); err != nil {
		t.Fatalf("%v", err)
	}

	// Deserialize
	e, err := Uint8(buf)
	if err != nil {
		t.Fatalf("%v", err)
	}
	f, err := Uint16(buf, binary.LittleEndian)
	if err != nil {
		t.Fatalf("%v", err)
	}
	g, err := Uint32(buf, binary.LittleEndian)
	if err != nil {
		t.Fatalf("%v", err)
	}
	h, err := Uint64(buf, binary.LittleEndian)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Compare
	assert.Equal(t, a, e)
	assert.Equal(t, b, f)
	assert.Equal(t, c, g)
	assert.Equal(t, d, h)
}

// Default serialization test, but with signed integers
func TestSignedInteger(t *testing.T) {
	a := int8(-5)
	b := int16(-10)
	c := int32(-20)
	d := int64(-40)

	// Serialize
	buf := new(bytes.Buffer)

	if err := PutUint8(buf, uint8(a)); err != nil {
		t.Fatalf("%v", err)
	}
	if err := PutUint16(buf, binary.LittleEndian, uint16(b)); err != nil {
		t.Fatalf("%v", err)
	}
	if err := PutUint32(buf, binary.LittleEndian, uint32(c)); err != nil {
		t.Fatalf("%v", err)
	}
	if err := PutUint64(buf, binary.LittleEndian, uint64(d)); err != nil {
		t.Fatalf("%v", err)
	}

	// Deserialize
	e, err := Uint8(buf)
	if err != nil {
		t.Fatalf("%v", err)
	}
	f, err := Uint16(buf, binary.LittleEndian)
	if err != nil {
		t.Fatalf("%v", err)
	}
	g, err := Uint32(buf, binary.LittleEndian)
	if err != nil {
		t.Fatalf("%v", err)
	}
	h, err := Uint64(buf, binary.LittleEndian)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Compare with type conversion
	assert.Equal(t, a, int8(e))
	assert.Equal(t, b, int16(f))
	assert.Equal(t, c, int32(g))
	assert.Equal(t, d, int64(h))
}

// Testing all the CompactSize formats
func TestCompactSize(t *testing.T) {
	a := uint64(1)
	b := uint64(2 ^ 8)
	c := uint64(2 ^ 16)
	d := uint64(2 ^ 32)

	// Serialize
	buf := new(bytes.Buffer)
	if err := WriteVarInt(buf, uint64(a)); err != nil {
		t.Fatalf("%v", err)
	}
	if err := WriteVarInt(buf, uint64(b)); err != nil {
		t.Fatalf("%v", err)
	}
	if err := WriteVarInt(buf, uint64(c)); err != nil {
		t.Fatalf("%v", err)
	}
	if err := WriteVarInt(buf, uint64(d)); err != nil {
		t.Fatalf("%v", err)
	}

	// Deserialize
	e, err := ReadVarInt(buf)
	if err != nil {
		t.Fatalf("%v", err)
	}
	f, err := ReadVarInt(buf)
	if err != nil {
		t.Fatalf("%v", err)
	}
	g, err := ReadVarInt(buf)
	if err != nil {
		t.Fatalf("%v", err)
	}
	h, err := ReadVarInt(buf)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Compare
	assert.Equal(t, a, e)
	assert.Equal(t, b, f)
	assert.Equal(t, c, g)
	assert.Equal(t, d, h)
}

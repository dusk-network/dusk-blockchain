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

	if err := WriteUint8(buf, a); err != nil {
		t.Fatal(err)
	}
	if err := WriteUint16(buf, binary.LittleEndian, b); err != nil {
		t.Fatal(err)
	}
	if err := WriteUint32(buf, binary.LittleEndian, c); err != nil {
		t.Fatal(err)
	}
	if err := WriteUint64(buf, binary.LittleEndian, d); err != nil {
		t.Fatal(err)
	}

	// Deserialize
	var e uint8
	var f uint16
	var g uint32
	var h uint64
	if err := ReadUint8(buf, &e); err != nil {
		t.Fatal(err)
	}
	if err := ReadUint16(buf, binary.LittleEndian, &f); err != nil {
		t.Fatal(err)
	}
	if err := ReadUint32(buf, binary.LittleEndian, &g); err != nil {
		t.Fatal(err)
	}
	if err := ReadUint64(buf, binary.LittleEndian, &h); err != nil {
		t.Fatal(err)
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

	if err := WriteUint8(buf, uint8(a)); err != nil {
		t.Fatal(err)
	}
	if err := WriteUint16(buf, binary.LittleEndian, uint16(b)); err != nil {
		t.Fatal(err)
	}
	if err := WriteUint32(buf, binary.LittleEndian, uint32(c)); err != nil {
		t.Fatal(err)
	}
	if err := WriteUint64(buf, binary.LittleEndian, uint64(d)); err != nil {
		t.Fatal(err)
	}

	// Deserialize
	var e uint8
	var f uint16
	var g uint32
	var h uint64
	if err := ReadUint8(buf, &e); err != nil {
		t.Fatal(err)
	}
	if err := ReadUint16(buf, binary.LittleEndian, &f); err != nil {
		t.Fatal(err)
	}
	if err := ReadUint32(buf, binary.LittleEndian, &g); err != nil {
		t.Fatal(err)
	}
	if err := ReadUint64(buf, binary.LittleEndian, &h); err != nil {
		t.Fatal(err)
	}

	// Compare with type conversion
	assert.Equal(t, a, int8(e))
	assert.Equal(t, b, int16(f))
	assert.Equal(t, c, int32(g))
	assert.Equal(t, d, int64(h))
}

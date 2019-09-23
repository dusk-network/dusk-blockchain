package encoding_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/stretchr/testify/assert"
)

//// Unit tests

// Default serialization test
func TestIntegerEncodeDecode(t *testing.T) {
	a := uint8(5)
	b := uint16(10)
	c := uint32(20)
	d := uint64(40)

	// Serialize
	buf := new(bytes.Buffer)

	if err := encoding.WriteUint8(buf, a); err != nil {
		t.Fatal(err)
	}
	if err := encoding.WriteUint16LE(buf, b); err != nil {
		t.Fatal(err)
	}
	if err := encoding.WriteUint32LE(buf, c); err != nil {
		t.Fatal(err)
	}
	if err := encoding.WriteUint64LE(buf, d); err != nil {
		t.Fatal(err)
	}

	// Deserialize
	var e uint8
	var f uint16
	var g uint32
	var h uint64
	if err := encoding.ReadUint8(buf, &e); err != nil {
		t.Fatal(err)
	}
	if err := encoding.ReadUint16LE(buf, &f); err != nil {
		t.Fatal(err)
	}
	if err := encoding.ReadUint32LE(buf, &g); err != nil {
		t.Fatal(err)
	}
	if err := encoding.ReadUint64LE(buf, &h); err != nil {
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

	if err := encoding.WriteUint8(buf, uint8(a)); err != nil {
		t.Fatal(err)
	}
	if err := encoding.WriteUint16LE(buf, uint16(b)); err != nil {
		t.Fatal(err)
	}
	if err := encoding.WriteUint32LE(buf, uint32(c)); err != nil {
		t.Fatal(err)
	}
	if err := encoding.WriteUint64LE(buf, uint64(d)); err != nil {
		t.Fatal(err)
	}

	// Deserialize
	var e uint8
	var f uint16
	var g uint32
	var h uint64
	if err := encoding.ReadUint8(buf, &e); err != nil {
		t.Fatal(err)
	}
	if err := encoding.ReadUint16LE(buf, &f); err != nil {
		t.Fatal(err)
	}
	if err := encoding.ReadUint32LE(buf, &g); err != nil {
		t.Fatal(err)
	}
	if err := encoding.ReadUint64LE(buf, &h); err != nil {
		t.Fatal(err)
	}

	// Compare with type conversion
	assert.Equal(t, a, int8(e))
	assert.Equal(t, b, int16(f))
	assert.Equal(t, c, int32(g))
	assert.Equal(t, d, int64(h))
}

//// Benchmarks

func BenchmarkReadUint8Interface(b *testing.B) {
	var v uint8

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer([]byte{1})
		if err := ReadUint8(buf, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadUint8NoInterface(b *testing.B) {
	var v uint8

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer([]byte{1})
		if err := encoding.ReadUint8(buf, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteUint8Interface(b *testing.B) {
	n := uint8(rand.Uint32())
	for i := 0; i < b.N; i++ {
		if err := WriteUint8(new(bytes.Buffer), n); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteUint8NoInterface(b *testing.B) {
	n := uint8(rand.Uint32())
	for i := 0; i < b.N; i++ {
		if err := encoding.WriteUint8(new(bytes.Buffer), n); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadUint16Interface(b *testing.B) {
	var v uint16

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer([]byte{1, 2})
		if err := ReadUint16(buf, binary.LittleEndian, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadUint16NoInterface(b *testing.B) {
	var v uint16

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer([]byte{1, 2})
		if err := encoding.ReadUint16LE(buf, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteUint16Interface(b *testing.B) {
	n := uint16(rand.Uint32())
	for i := 0; i < b.N; i++ {
		if err := WriteUint16(new(bytes.Buffer), binary.LittleEndian, n); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteUint16NoInterface(b *testing.B) {
	n := uint16(rand.Uint32())
	for i := 0; i < b.N; i++ {
		if err := encoding.WriteUint16LE(new(bytes.Buffer), n); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadUint32Interface(b *testing.B) {
	var v uint32

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer([]byte{1, 2, 3, 4})
		if err := ReadUint32(buf, binary.LittleEndian, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadUint32NoInterface(b *testing.B) {
	var v uint32

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer([]byte{1, 2, 3, 4})
		if err := encoding.ReadUint32LE(buf, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteUint32Interface(b *testing.B) {
	n := rand.Uint32()
	for i := 0; i < b.N; i++ {
		if err := WriteUint32(new(bytes.Buffer), binary.LittleEndian, n); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteUint32NoInterface(b *testing.B) {
	n := rand.Uint32()
	for i := 0; i < b.N; i++ {
		if err := encoding.WriteUint32LE(new(bytes.Buffer), n); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadUint64Interface(b *testing.B) {
	var v uint64

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer([]byte{1, 2, 3, 4, 5, 6, 7, 8})
		if err := ReadUint64(buf, binary.LittleEndian, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadUint64NoInterface(b *testing.B) {
	var v uint64

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer([]byte{1, 2, 3, 4, 5, 6, 7, 8})
		if err := encoding.ReadUint64LE(buf, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteUint64Interface(b *testing.B) {
	n := rand.Uint64()
	for i := 0; i < b.N; i++ {
		if err := WriteUint64(new(bytes.Buffer), binary.LittleEndian, n); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteUint64NoInterface(b *testing.B) {
	n := rand.Uint64()
	for i := 0; i < b.N; i++ {
		if err := encoding.WriteUint64LE(new(bytes.Buffer), n); err != nil {
			b.Fatal(err)
		}
	}
}

//// Old functions, which make use of interfaces
//// Purely here for benchmark purposes

// ReadUint8 will read a single byte and put it into v.
func ReadUint8(r io.Reader, v *uint8) error {
	var b [1]byte
	if _, err := r.Read(b[:]); err != nil {
		return err
	}
	// Since we only read one byte, it will either succeed
	// or throw an EOF error. Thus, we don't check the number of
	// bytes read here.
	*v = b[0]
	return nil
}

// ReadUint16 will read two bytes and convert them to a uint16
// from the specified byte order. The result is put into v.
func ReadUint16(r io.Reader, o binary.ByteOrder, v *uint16) error {
	var b [2]byte
	n, err := r.Read(b[:])
	if err != nil || n != len(b) {
		return fmt.Errorf("encoding: ReadUint16 read %v/2 bytes - %v", n, err)
	}
	*v = o.Uint16(b[:])
	return nil
}

// ReadUint32 will read four bytes and convert them to a uint32
// from the specified byte order. The result is put into v.
func ReadUint32(r io.Reader, o binary.ByteOrder, v *uint32) error {
	var b [4]byte
	n, err := r.Read(b[:])
	if err != nil || n != len(b) {
		return fmt.Errorf("encoding: ReadUint32 read %v/4 bytes - %v", n, err)
	}
	*v = o.Uint32(b[:])
	return nil
}

// ReadUint64 will read eight bytes and convert them to a uint64
// from the specified byte order. The result is put into v.
func ReadUint64(r io.Reader, o binary.ByteOrder, v *uint64) error {
	var b [8]byte
	n, err := r.Read(b[:])
	if err != nil || n != len(b) {
		return fmt.Errorf("encoding: ReadUint64 read %v/8 bytes - %v", n, err)
	}
	*v = o.Uint64(b[:])
	return nil
}

// WriteUint8 will write a single byte.
func WriteUint8(w io.Writer, v uint8) error {
	var b [1]byte
	b[0] = v
	_, err := w.Write(b[:])
	return err
}

// WriteUint16 will write two bytes in the specified byte order.
func WriteUint16(w io.Writer, o binary.ByteOrder, v uint16) error {
	var b [2]byte
	o.PutUint16(b[:], v)
	_, err := w.Write(b[:])
	return err
}

// WriteUint32 will write four bytes in the specified byte order.
func WriteUint32(w io.Writer, o binary.ByteOrder, v uint32) error {
	var b [4]byte
	o.PutUint32(b[:], v)
	_, err := w.Write(b[:])
	return err
}

// WriteUint64 will write eight bytes in the specified byte order.
func WriteUint64(w io.Writer, o binary.ByteOrder, v uint64) error {
	var b [8]byte
	o.PutUint64(b[:], v)
	_, err := w.Write(b[:])
	return err
}

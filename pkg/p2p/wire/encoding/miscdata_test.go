// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package encoding_test

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

//// Unit tests

// Test bool functions.
func TestBoolEncodeDecode(t *testing.T) {
	b := true

	buf := new(bytes.Buffer)
	if err := encoding.WriteBool(buf, b); err != nil {
		t.Fatal(err)
	}

	var c bool
	if err := encoding.ReadBool(buf, &c); err != nil {
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
	if err := encoding.Write256(buf, byte32); err != nil {
		t.Fatal(err)
	}

	// Check if it serialized correctly
	assert.Equal(t, buf.Bytes(), byte32)
	assert.Equal(t, len(buf.Bytes()), 32)

	// Deserialize
	hash := make([]byte, 32)
	if err := encoding.Read256(buf, hash); err != nil {
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
	if err := encoding.Write512(buf, byte64); err != nil {
		t.Fatal(err)
	}

	// Check if it serialized correctly
	assert.Equal(t, buf.Bytes(), byte64)
	assert.Equal(t, len(buf.Bytes()), 64)

	// Deserialize
	hash := make([]byte, 64)
	if err := encoding.Read512(buf, hash); err != nil {
		t.Fatal(err)
	}

	// Content should be the same
	assert.Equal(t, hash, byte64)
}

// Test to make sure it only takes byte slices of length 32.
func Test256Length(t *testing.T) {
	byte16, err := crypto.RandEntropy(16)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize
	buf := new(bytes.Buffer)

	err = encoding.Write256(buf, byte16) // This should fail
	if err == nil {
		t.Fatal("did not throw error when serializing byte slice of improper length")
	}

	buf.Reset()

	byte40, err := crypto.RandEntropy(40)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize
	err = encoding.Write256(buf, byte40) // This should also fail
	if err == nil {
		t.Fatal("did not throw error when serializing byte slice of improper length")
	}
}

// Test to make sure it only takes byte slices of length 64.
func Test512Length(t *testing.T) {
	byte16, err := crypto.RandEntropy(16)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize
	buf := new(bytes.Buffer)

	err = encoding.Write512(buf, byte16) // This should fail
	if err == nil {
		t.Fatal("did not throw error when serializing byte slice of improper length")
	}

	buf.Reset()

	byte80, err := crypto.RandEntropy(80)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize
	err = encoding.Write512(buf, byte80) // This should also fail
	if err == nil {
		t.Fatal("did not throw error when serializing byte slice of improper length")
	}
}

//// Benchmarks

func BenchmarkReadBoolInterface(b *testing.B) {
	var t bool

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer([]byte{1})
		if err := ReadBool(buf, &t); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadBoolNoInterface(b *testing.B) {
	var t bool

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer([]byte{1})
		if err := encoding.ReadBool(buf, &t); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteBoolInterface(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := WriteBool(new(bytes.Buffer), true); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteBoolNoInterface(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := encoding.WriteBool(new(bytes.Buffer), true); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRead256Interface(b *testing.B) {
	var bs []byte

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(make([]byte, 32))
		if err := Read256(buf, &bs); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRead256NoInterface(b *testing.B) {
	bs := make([]byte, 32)

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(make([]byte, 32))
		if err := encoding.Read256(buf, bs); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWrite256Interface(b *testing.B) {
	bs := make([]byte, 32)

	for i := 0; i < b.N; i++ {
		if err := Write256(new(bytes.Buffer), bs); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWrite256NoInterface(b *testing.B) {
	bs := make([]byte, 32)

	for i := 0; i < b.N; i++ {
		if err := encoding.Write256(new(bytes.Buffer), bs); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRead512Interface(b *testing.B) {
	var bs []byte

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(make([]byte, 64))
		if err := Read512(buf, &bs); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRead512NoInterface(b *testing.B) {
	bs := make([]byte, 64)

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(make([]byte, 64))
		if err := encoding.Read512(buf, bs); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWrite512Interface(b *testing.B) {
	bs := make([]byte, 64)

	for i := 0; i < b.N; i++ {
		if err := Write512(new(bytes.Buffer), bs); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWrite512NoInterface(b *testing.B) {
	bs := make([]byte, 64)

	for i := 0; i < b.N; i++ {
		if err := encoding.Write512(new(bytes.Buffer), bs); err != nil {
			b.Fatal(err)
		}
	}
}

// TODO: add benchmark for BLS

//// Old functions which make uses of interfaces
//// Here for benchmarking purposes only

// ReadBool will read a single byte from r, turn it into a bool
// and pass it into v.
func ReadBool(r io.Reader, b *bool) error {
	var v uint8
	if err := ReadUint8(r, &v); err != nil {
		return err
	}

	*b = v != 0
	return nil
}

// WriteBool will write a boolean value as a single byte into w.
func WriteBool(w io.Writer, b bool) error {
	v := uint8(0)
	if b {
		v = 1
	}

	return WriteUint8(w, v)
}

// Read256 will read 32 bytes from r into b.
func Read256(r io.Reader, b *[]byte) error {
	*b = make([]byte, 32)

	n, err := r.Read(*b)
	if err != nil || n != len(*b) {
		return fmt.Errorf("encoding: Read256 read %v/32 bytes - %v", n, err)
	}

	return nil
}

// Write256 will check the length of b and then write to w.
func Write256(w io.Writer, b []byte) error {
	if len(b) != 32 {
		return fmt.Errorf("b is not proper size - expected 32 bytes, is actually %d bytes", len(b))
	}

	if _, err := w.Write(b); err != nil {
		return err
	}

	return nil
}

// Read512 will read 64 bytes from r into b.
func Read512(r io.Reader, b *[]byte) error {
	*b = make([]byte, 64)

	n, err := r.Read(*b)
	if err != nil || n != len(*b) {
		return fmt.Errorf("encoding: Read512 read %v/64 bytes - %v", n, err)
	}

	return nil
}

// Write512 will check the length of b and then write to w.
func Write512(w io.Writer, b []byte) error {
	if len(b) != 64 {
		return fmt.Errorf("b is not proper size - expected 64 bytes, is actually %d bytes", len(b))
	}

	if _, err := w.Write(b); err != nil {
		return err
	}

	return nil
}

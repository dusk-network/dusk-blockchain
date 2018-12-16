package transactions

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gitlab.dusk.network/dusk-core/dusk-go/crypto"
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
		t.Fatalf("%v", err)
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
		t.Fatalf("%v", err)
	}

	// Deserialize
	var newStealth Stealth

	if err := newStealth.Decode(buf); err != nil {
		t.Fatalf("%v", err)
	}

	// Compare
	assert.Equal(t, s, newStealth)
}

// Benchmark the speed of the dusk encoding lib vs the standard encoding lib
// This benchmark will show the time differences along with a percentual
// difference at the end. It can vary from run to run, but will roughly
// come out to about 30% decrease in time needed.
func BenchmarkCompare(b *testing.B) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	// Input
	sig, _ := crypto.RandEntropy(2000)
	in := Input{
		byte32,
		byte32,
		1,
		sig,
	}

	in2 := Input{
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

	out2 := Output{
		500,
		byte32,
	}

	// Type attribute
	ta := TypeAttributes{
		[]Input{in, in2},
		byte32,
		[]Output{out, out2},
	}

	R, _ := crypto.RandEntropy(32)
	s := Stealth{
		1,
		1,
		R,
		ta,
	}

	// standard encoding lib
	runs := 1000000 // Amount of txs to serialize
	start1 := time.Now()
	for i := 0; i < runs; i++ {
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, s.Version)
		binary.Write(buf, binary.LittleEndian, s.Type)
		binary.Write(buf, binary.LittleEndian, s.R)
		binary.Write(buf, binary.LittleEndian, len(s.TA.Inputs))
		for _, input := range s.TA.Inputs {
			binary.Write(buf, binary.LittleEndian, input.KeyImage)
			binary.Write(buf, binary.LittleEndian, input.TxID)
			binary.Write(buf, binary.LittleEndian, input.Index)
			binary.Write(buf, binary.LittleEndian, input.Signature)
		}

		binary.Write(buf, binary.LittleEndian, s.TA.TxPubKey)
		binary.Write(buf, binary.LittleEndian, len(s.TA.Outputs))
		for _, output := range s.TA.Outputs {
			binary.Write(buf, binary.LittleEndian, output.Amount)
			binary.Write(buf, binary.LittleEndian, output.P)
		}
	}

	end1 := time.Now()
	elapsed1 := end1.Sub(start1)
	b.Logf("%.2f", elapsed1.Seconds())

	// dusk encoding lib
	start2 := time.Now()
	for i := 0; i < runs; i++ {
		size := s.GetEncodeSize()
		bs := make([]byte, 0, size)
		buf := bytes.NewBuffer(bs)

		if err := s.Encode(buf); err != nil {
			b.Fatalf("%v", err)
		}
	}

	end2 := time.Now()
	elapsed2 := end2.Sub(start2)
	b.Logf("%.2f", elapsed2.Seconds())

	div := elapsed1.Seconds() / elapsed2.Seconds()
	increase := (div - 1) * 100
	b.Logf("%.1f%% performance increase when using dusk lib", increase)
}

// Benchmark the dusk encoding lib (to see benchmark stats for this approach)
func BenchmarkDuskLib(b *testing.B) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	// Input
	sig, _ := crypto.RandEntropy(2000)
	in := Input{
		byte32,
		byte32,
		1,
		sig,
	}

	in2 := Input{
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

	out2 := Output{
		500,
		byte32,
	}

	// Type attribute
	ta := TypeAttributes{
		[]Input{in, in2},
		byte32,
		[]Output{out, out2},
	}

	R, _ := crypto.RandEntropy(32)
	s := Stealth{
		1,
		1,
		R,
		ta,
	}

	for i := 0; i < 1000000; i++ {
		size := s.GetEncodeSize()
		bs := make([]byte, 0, size)
		buf := bytes.NewBuffer(bs)

		if err := s.Encode(buf); err != nil {
			b.Fatalf("%v", err)
		}
	}
}

// Benchmark the standard encoding lib (to see benchmark stats for this approach)
func BenchmarkStandardLib(b *testing.B) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	// Input
	sig, _ := crypto.RandEntropy(2000)
	in := Input{
		byte32,
		byte32,
		1,
		sig,
	}

	in2 := Input{
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

	out2 := Output{
		500,
		byte32,
	}

	// Type attribute
	ta := TypeAttributes{
		[]Input{in, in2},
		byte32,
		[]Output{out, out2},
	}

	R, _ := crypto.RandEntropy(32)
	s := Stealth{
		1,
		1,
		R,
		ta,
	}

	for i := 0; i < 1000000; i++ {
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, s.Version)
		binary.Write(buf, binary.LittleEndian, s.Type)
		binary.Write(buf, binary.LittleEndian, s.R)
		binary.Write(buf, binary.LittleEndian, len(s.TA.Inputs))
		for _, input := range s.TA.Inputs {
			binary.Write(buf, binary.LittleEndian, input.KeyImage)
			binary.Write(buf, binary.LittleEndian, input.TxID)
			binary.Write(buf, binary.LittleEndian, input.Index)
			binary.Write(buf, binary.LittleEndian, input.Signature)
		}

		binary.Write(buf, binary.LittleEndian, s.TA.TxPubKey)
		binary.Write(buf, binary.LittleEndian, len(s.TA.Outputs))
		for _, output := range s.TA.Outputs {
			binary.Write(buf, binary.LittleEndian, output.Amount)
			binary.Write(buf, binary.LittleEndian, output.P)
		}
	}
}

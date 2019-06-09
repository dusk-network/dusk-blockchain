package processing

import (
	"bytes"
	"testing"
)

func TestCOBS(t *testing.T) {

	var input bytes.Buffer
	input.Write([]byte{1, 2, 3, 4})
	decoded := Decode(Encode(&input).Bytes())

	if bytes.Equal(input.Bytes(), decoded.Bytes()) == false {
		t.Fatal("cobs scenario 1 failed")
	}
}

func TestCOBSPacket(t *testing.T) {

	var input bytes.Buffer
	var i byte = 1
	for ; i <= 0xfe; i++ {
		input.WriteByte(i)
	}

	decoded := Decode(Encode(&input).Bytes())

	if bytes.Equal(input.Bytes(), decoded.Bytes()) == false {
		t.Fatal("cobs scenario 2 failed")
	}
}

func TestCOBSZeros(t *testing.T) {

	var input bytes.Buffer
	var i byte = 1
	for ; i <= 3; i++ {
		input.WriteByte(0)
	}

	decoded := Decode(Encode(&input).Bytes())

	if bytes.Equal(input.Bytes(), decoded.Bytes()) == false {
		t.Fatal("cobs scenario 3 failed")
	}
}

func TestCOBSPacketZero1(t *testing.T) {

	var input bytes.Buffer
	var i byte = 1
	for ; i <= 0xfe; i++ {
		input.WriteByte(i)
	}

	// extra zero
	input.WriteByte(0)

	decoded := Decode(Encode(&input).Bytes())

	if bytes.Equal(input.Bytes(), decoded.Bytes()) == false {
		t.Fatal("cobs scenario 4 failed")
	}
}
func TestCOBSPacketZero2(t *testing.T) {

	var input bytes.Buffer
	var i byte = 1
	for ; i <= 0xfe; i++ {
		input.WriteByte(i)
	}

	i = 1
	for ; i <= 3; i++ {
		input.WriteByte(i)
	}

	// extra zero
	input.WriteByte(0)

	decoded := Decode(Encode(&input).Bytes())

	if bytes.Equal(input.Bytes(), decoded.Bytes()) == false {
		t.Fatal("cobs scenario 5 failed")
	}
}

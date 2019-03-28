package zkproof

import (
	"encoding/binary"
	"errors"
	"io/ioutil"
	"syscall"
)

// BytesArray is a very basic structure that reflects the one used in the Rust process.
type BytesArray struct {
	bytes []byte
	index int
}

// Bytes method returns the bytes hold in the BytesArray struct
func (ba *BytesArray) Bytes() []byte {
	return ba.bytes
}

// ReadUint8 reads the next byte in the BytesArray
func (ba *BytesArray) ReadUint8() (uint8, error) {
	if ba.index >= len(ba.bytes) {
		return 0, errors.New("Index overflow")
	}
	value := ba.bytes[ba.index]
	ba.index++
	return value, nil
}

// ReadUint32 reads the next 4 bytes in the BytesArray as uint32
func (ba *BytesArray) ReadUint32() (uint32, error) {
	if ba.index >= len(ba.bytes) {
		return 0, errors.New("Index overflow")
	}
	length := binary.LittleEndian.Uint32(ba.bytes[ba.index:])
	ba.index += 4

	return length, nil
}

// WriteUint8 writes a byte into a BytesArray struct
func (ba *BytesArray) WriteUint8(value uint8) {
	ba.bytes = append(ba.bytes, value)
}

// WriteUint32 writes a uint32 into a BytesArray struct
func (ba *BytesArray) WriteUint32(v uint32) {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, v)

	ba.bytes = append(ba.bytes, bytes...)
}

func (ba *BytesArray) Read() []byte {
	length, err := ba.ReadUint32()
	if err != nil {
		return nil
	}

	bytes := ba.bytes[ba.index : ba.index+int(length)]
	ba.index += int(length)

	return bytes
}

func (ba *BytesArray) Write(bytes []byte) {
	length := uint32(len(bytes))
	ba.WriteUint32(length)

	ba.bytes = append(ba.bytes, bytes...)
}

// NamedPipe holds data for a named pipe
type NamedPipe struct {
	Path string
}

// NewNamedPipe constructs a new named pipe using the path given
func NewNamedPipe(path string) NamedPipe {
	np := NamedPipe{Path: path}
	syscall.Mkfifo(path, 0644)
	return np
}

// ReadBytes consumes and return all the bytes in the named pipe
func (p *NamedPipe) ReadBytes() (BytesArray, error) {
	bytes, err := ioutil.ReadFile(p.Path)

	return BytesArray{bytes: bytes, index: 0}, err
}

// WriteBytes write a BytesArray into the named pipe
func (p *NamedPipe) WriteBytes(ba BytesArray) error {
	return ioutil.WriteFile(p.Path, ba.Bytes(), 0644)
}

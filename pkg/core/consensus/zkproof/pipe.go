package zkproof

import (
	"encoding/binary"
	"errors"
	"io/ioutil"
	"os"
)

type BytesArray struct {
	bytes []byte
	// this shouldn't be public
	index int
}

func (ba *BytesArray) Bytes() []byte {
	return ba.bytes
}

func (ba *BytesArray) ReadUint8() (uint8, error) {
	if ba.index >= len(ba.bytes) {
		return 0, errors.New("Index overflow")
	}
	value := ba.bytes[ba.index]
	ba.index++
	return value, nil
}

func (ba *BytesArray) ReadUint32() (uint32, error) {
	if ba.index >= len(ba.bytes) {
		return 0, errors.New("Index overflow")
	}
	length := binary.LittleEndian.Uint32(ba.bytes[ba.index:])
	ba.index += 4

	return length, nil
}

func (ba *BytesArray) WriteUint8(value uint8) {
	ba.bytes = append(ba.bytes, value)
}

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

func (p *NamedPipe) ReadBytes() (BytesArray, error) {
	bytes, err := ioutil.ReadFile(p.Path)

	return BytesArray{bytes: bytes, index: 0}, err
}

func (p *NamedPipe) WriteBytes(ba BytesArray) error {
	f, err := os.OpenFile(p.Path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	defer f.Close()
	if err != nil {
		return err
	}
	nb, err := f.Write(ba.Bytes())
	if err != nil {
		return err
	}
	if nb != len(ba.Bytes()) {
		return errors.New("lenghts don't match")
	}
	return nil
}

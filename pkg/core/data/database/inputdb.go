package database

import (
	"encoding/binary"
	"io"

	"github.com/bwesterb/go-ristretto"
)

type inputDB struct {
	amount, mask, privKey ristretto.Scalar
	unlockHeight          uint64
}

func (idb *inputDB) Decode(r io.Reader) error {
	amountBytes, err := read32Bytes(r)
	if err != nil {
		return err
	}
	idb.amount.SetBytes(&amountBytes)

	maskBytes, err := read32Bytes(r)
	if err != nil {
		return err
	}
	idb.mask.SetBytes(&maskBytes)

	privKeyBytes, err := read32Bytes(r)
	if err != nil {
		return err
	}
	idb.privKey.SetBytes(&privKeyBytes)

	var unlockHeight uint64
	err = binary.Read(r, binary.LittleEndian, &unlockHeight)
	if err != nil {
		return err
	}
	idb.unlockHeight = unlockHeight

	return nil
}

func (idb *inputDB) Encode(w io.Writer) error {
	err := binary.Write(w, binary.BigEndian, idb.amount.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, idb.mask.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, idb.privKey.Bytes())
	if err != nil {
		return err
	}

	return binary.Write(w, binary.LittleEndian, idb.unlockHeight)
}

func read32Bytes(r io.Reader) ([32]byte, error) {
	x := [32]byte{}
	err := binary.Read(r, binary.BigEndian, &x)
	if err != nil {
		return x, err
	}
	return x, nil
}

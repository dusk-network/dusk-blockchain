package database

import (
	"encoding/binary"
	"io"

	"github.com/bwesterb/go-ristretto"
)

type inputDB struct {
	amount, mask, privKey ristretto.Scalar
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

	return nil
}

func read32Bytes(r io.Reader) ([32]byte, error) {
	x := [32]byte{}
	err := binary.Read(r, binary.BigEndian, &x)
	if err != nil {
		return x, err
	}
	return x, nil
}

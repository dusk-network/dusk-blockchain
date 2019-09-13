package protocol

import (
	"bytes"
	"strconv"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

// Version is a struct that separates version fields.
type Version struct {
	Major uint8
	Minor uint8
	Patch uint16
}

func (v Version) String() string {
	return strconv.Itoa(int(v.Major)) + "." + strconv.Itoa(int(v.Minor)) + "." + strconv.Itoa(int(v.Patch))
}

// Encode will encode a Version struct to w.
func (v *Version) Encode(w *bytes.Buffer) error {
	if err := encoding.WriteUint8(w, v.Major); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, v.Minor); err != nil {
		return err
	}

	if err := encoding.WriteUint16LE(w, v.Patch); err != nil {
		return err
	}

	return nil
}

// Decode will Decodde a Version struct from r.
func (v *Version) Decode(r *bytes.Buffer) error {
	var err error
	v.Major, err = encoding.ReadUint8(r)
	if err != nil {
		return err
	}

	v.Minor, err = encoding.ReadUint8(r)
	if err != nil {
		return err
	}

	v.Patch, err = encoding.ReadUint16LE(r)
	if err != nil {
		return err
	}

	return nil
}

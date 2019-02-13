package protocol

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Version is a struct that seperates version fields.
type Version struct {
	Major uint64
	Minor uint64
	Patch uint64
}

// Encode will encode a Version struct to w.
func (v *Version) Encode(w io.Writer) error {
	if err := encoding.WriteUint64(w, binary.LittleEndian, v.Major); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, v.Minor); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, v.Patch); err != nil {
		return err
	}

	return nil
}

// Decode will Decodde a Version struct from r.
func (v *Version) Decode(r io.Reader) error {
	if err := encoding.ReadUint64(r, binary.LittleEndian, &v.Major); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &v.Minor); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &v.Patch); err != nil {
		return err
	}

	return nil
}

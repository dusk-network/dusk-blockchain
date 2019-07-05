package mlsag

import (
	"encoding/binary"
	"errors"
	"io"

	ristretto "github.com/bwesterb/go-ristretto"
)

type PubKeys struct {
	// Set to true if the set of pubKeys are decoys
	decoy bool
	// Vector of pubKeys
	keys []ristretto.Point
}

func (p *PubKeys) Equals(other PubKeys) bool {

	if len(p.keys) != len(other.keys) {
		return false
	}

	for i := range p.keys {
		ok := p.keys[i].Equals(&other.keys[i])
		if !ok {
			return ok
		}
	}

	return true
}

func (p PubKeys) Encode(w io.Writer) error {
	for i := range p.keys {
		err := binary.Write(w, binary.BigEndian, p.keys[i].Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PubKeys) Decode(r io.Reader, numKeys uint32) error {

	if p == nil {
		return errors.New("struct is nil")
	}

	var xBytes [32]byte
	var x ristretto.Point
	for i := uint32(0); i < numKeys; i++ {
		err := binary.Read(r, binary.BigEndian, &xBytes)
		if err != nil {
			return err
		}
		ok := x.SetBytes(&xBytes)
		if !ok {
			return errors.New("point not encodable")
		}
		p.AddPubKey(x)
	}
	return nil
}
func (p *PubKeys) AddPubKey(key ristretto.Point) {
	p.keys = append(p.keys, key)
}

func (p *PubKeys) Len() int {
	return len(p.keys)
}

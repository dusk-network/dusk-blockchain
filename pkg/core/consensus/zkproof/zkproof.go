package zkproof

import (
	"bytes"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

// Zkproof is the zkproof generated for the block generation procedure
// XXX; STUB
type Zkproof struct {
}

// Prove generates the zero-knowledge proof of the blind-bid
func Prove(X, Y, Z, M, k []byte, Q, d uint64) *Zkproof {
	return &Zkproof{}
}

// Verify verifies the correctness of the zk proof
func (z *Zkproof) Verify() bool {
	return true
}

// Encode serialises the zk proof into a io.Writer interface
func (z *Zkproof) Encode(w io.Writer) error {

	randData, _ := crypto.RandEntropy(40)
	_, err := w.Write(randData)
	if err != nil {
		return err
	}

	return nil
}

// Bytes serialises the zk proof into a byte slice
// and returns the byte slice
func (z *Zkproof) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := z.Encode(buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode serialises the zk proof from a byte slice
func (z *Zkproof) Decode(r io.Reader) error {
	return nil
}

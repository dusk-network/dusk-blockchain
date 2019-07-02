package transactions

import (
	"bytes"
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Standard is a generic transaction. It can also be seen as a stealth transaction.
// It is used to make basic payments on the dusk network.
type Standard struct {
	// TxType represents the transaction type
	TxType TxType
	// R is the Transaction Public Key
	R []byte
	// Version is the transaction version. It does not use semver.
	// A new transaction version denotes a modification of the previous structure
	Version uint8 // 1 byte
	// TxID is the hash of the transaction fields.
	TxID []byte // TxID (32 bytes)
	// Inputs represent a list of inputs to the transaction.
	Inputs Inputs
	// Outputs represent a list of outputs to the transaction
	Outputs Outputs
	// Fee is the assosciated fee attached to the transaction. This is in cleartext.
	Fee uint64
	// RangeProof is the bulletproof rangeproof that proves that the hidden amount
	// is between 0 and 2^64
	RangeProof []byte // Variable size
}

// NewStandard will return a Standard transaction
// given the tx version and the fee atached
func NewStandard(ver uint8, fee uint64, R []byte) *Standard {
	return &Standard{
		R:       R,
		Version: ver,
		Fee:     fee,
		TxType:  StandardType,
	}
}

// AddInput will add an input to the list of inputs in the transaction.
func (s *Standard) AddInput(input *Input) {
	s.Inputs = append(s.Inputs, input)
}

// AddOutput will add an output to the list of outputs in the transaction.
func (s *Standard) AddOutput(output *Output) {
	s.Outputs = append(s.Outputs, output)
}

// Encode a Standard transaction and write it to an io.Writer
func (s *Standard) Encode(w io.Writer) error {

	if err := encoding.WriteUint8(w, uint8(s.TxType)); err != nil {
		return err
	}

	if err := encoding.Write256(w, s.R); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, uint8(s.Version)); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(w, uint64(len(s.Inputs))); err != nil {
		return err
	}

	for _, input := range s.Inputs {
		if err := input.Encode(w); err != nil {
			return err
		}
	}
	if err := encoding.WriteVarInt(w, uint64(len(s.Outputs))); err != nil {
		return err
	}

	for _, output := range s.Outputs {
		if err := output.Encode(w); err != nil {
			return err
		}
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, s.Fee); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, s.RangeProof); err != nil {
		return err
	}

	return nil
}

// Decode a reader into a standard transaction struct.
func (s *Standard) Decode(r io.Reader) error {

	var Type uint8
	if err := encoding.ReadUint8(r, &Type); err != nil {
		return err
	}
	s.TxType = TxType(Type)

	if err := encoding.Read256(r, &s.R); err != nil {
		return err
	}

	var ver uint8
	if err := encoding.ReadUint8(r, &ver); err != nil {
		return err
	}

	s.Version = ver

	lInputs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	s.Inputs = make(Inputs, lInputs)
	for i := uint64(0); i < lInputs; i++ {
		s.Inputs[i] = &Input{}
		if err := s.Inputs[i].Decode(r); err != nil {
			return err
		}
	}

	lOutputs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	s.Outputs = make(Outputs, lOutputs)
	for i := uint64(0); i < lOutputs; i++ {
		s.Outputs[i] = &Output{}
		if err := s.Outputs[i].Decode(r); err != nil {
			return err
		}
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &s.Fee); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.RangeProof); err != nil {
		return err
	}
	return nil
}

// CalculateHash hashes all of the encoded fields in a tx, if this has not been done already.
// The resulting byte array is also it's identifier
//// Implements merkletree.Payload interface
func (s *Standard) CalculateHash() ([]byte, error) {
	if len(s.TxID) != 0 {
		return s.TxID, nil
	}

	txid, err := hashBytes(s.Encode)
	if err != nil {
		return nil, err
	}
	s.TxID = txid

	return s.TxID, nil
}

// Type returns the associated TxType for the Standard struct.
// Implements the TypeInfo interface
func (s *Standard) Type() TxType {
	return s.TxType
}

// StandardTX returns the standard struct
// Implements Transaction interface.
func (s Standard) StandardTX() Standard {
	return s
}

// Equals returns true if two standard tx's are the same
func (s *Standard) Equals(t Transaction) bool {

	other, ok := t.(*Standard)
	if !ok {
		return false
	}

	if s.Version != other.Version {
		return false
	}

	if !bytes.Equal(s.R, other.R) {
		return false
	}

	if !s.Inputs.Equals(other.Inputs) {
		return false
	}

	if !s.Outputs.Equals(other.Outputs) {
		return false
	}

	if s.Fee != other.Fee {
		return false
	}

	// TxID is not compared, as this could be nil (not set)
	// We could check whether it is set and then check, but
	// the txid is not updated, if a modification is made after
	// calculating the hash. What we can do, is state this edge case and analyse our use-cases.

	return bytes.Equal(s.RangeProof, other.RangeProof)
}

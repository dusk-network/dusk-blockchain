package transactions

import (
	"bytes"
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

// CalculateHash hashes all of the encoded fields in a tx, if this has not been done already.
// The resulting byte array is also it's identifier
//// Implements merkletree.Payload interface
func (s *Standard) CalculateHash() ([]byte, error) {
	if len(s.TxID) != 0 {
		return s.TxID, nil
	}

	buf := new(bytes.Buffer)
	if err := MarshalStandard(buf, s); err != nil {
		return nil, err
	}

	txid, err := hashBytes(buf)
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

package transactions

import (
	"bytes"
	"errors"
	"io"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Stake transactions are time lock transactions that are used by nodes
// to allow them to become provisioners, and participate in the consensus protocol.
// What does the TimeLock represent?
// In a `Stake TX` it means that the Outputs, or the node's stake is locked until after the
// specified time. After this time, they will also no longer be a provisioner,
// and must re-stake to participate again.
type Stake struct {
	TimeLock
	PubKeyEd  []byte // 32 bytes
	PubKeyBLS []byte // 33 bytes
}

// NewStake will return a Stake transaction
// Given the tx version, the locktime,the fee and M
func NewStake(ver uint8, lock, fee uint64, R, pubKeyEd, pubKeyBLS []byte) (*Stake, error) {
	if len(pubKeyEd) != 32 {
		return nil, errors.New("edwards public key is not 32 bytes")
	}

	s := &Stake{
		TimeLock:  *NewTimeLock(ver, lock, fee, R),
		PubKeyEd:  pubKeyEd,
		PubKeyBLS: pubKeyBLS,
	}
	s.TxType = StakeType

	return s, nil
}

// Encode implements the Encoder interface
func (s *Stake) Encode(w io.Writer) error {
	if err := s.TimeLock.Encode(w); err != nil {
		return err
	}
	if err := encoding.Write256(w, s.PubKeyEd); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, s.PubKeyBLS); err != nil {
		return err
	}
	return nil
}

// Decode implements the Decoder interface
func (s *Stake) Decode(r io.Reader) error {

	if err := s.TimeLock.Decode(r); err != nil {
		return err
	}

	if err := encoding.Read256(r, &s.PubKeyEd); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.PubKeyBLS); err != nil {
		return err
	}
	return nil
}

// StandardTX returns the embedded standard tx
// Implements Transaction interface.
func (s Stake) StandardTX() Standard {
	return s.Standard
}

// CalculateHash hashes all of the encoded fields in a tx, if this has not been done already.
// The resulting byte array is also it's identifier
// Implements merkletree.Payload interface
func (s *Stake) CalculateHash() ([]byte, error) {
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

// Equals returns true if two Stake tx's are the same
func (s *Stake) Equals(t Transaction) bool {

	other, ok := t.(*Stake)
	if !ok {
		return false
	}

	if !s.Standard.Equals(&other.Standard) {
		return false
	}

	if !bytes.Equal(s.PubKeyEd, other.PubKeyEd) {
		return false
	}
	if !bytes.Equal(s.PubKeyBLS, other.PubKeyBLS) {
		return false
	}

	return true
}

func (s *Stake) GetOutputAmount() uint64 {
	var sAmount ristretto.Scalar
	sAmount.UnmarshalBinary(s.Outputs[0].EncryptedAmount)
	amount := sAmount.BigInt().Uint64()
	return amount
}

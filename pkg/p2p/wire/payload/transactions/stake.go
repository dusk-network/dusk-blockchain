package transactions

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Stake defines the transaction info for a staking transaction.
type Stake struct {
	PubKeyEd  []byte
	PubKeyBLS []byte
}

// NewStake will return a Stake struct with the passed public keys.
func NewStake(pubKeyEd, pubKeyBLS []byte) *Stake {
	return &Stake{
		PubKeyEd:  pubKeyEd,
		PubKeyBLS: pubKeyBLS,
	}
}

// Encode a Stake struct and write to w.
// Implements TypeInfo interface.
func (s *Stake) Encode(w io.Writer) error {
	if err := encoding.Write256(w, s.PubKeyEd); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, s.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Decode a Stake struct from r.
// Implements TypeInfo interface.
func (s *Stake) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &s.PubKeyEd); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Type returns the associated TxType for the Stake struct.
// Implements TypeInfo interface.
func (s *Stake) Type() TxType {
	return StakeType
}

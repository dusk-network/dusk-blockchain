package consensusmsg

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// BlockReduction defines a blockreduction message on the Dusk wire protocol.
type BlockReduction struct {
	BlockHash []byte // Hash of the block being voted on (32 bytes)
	SigBLS    []byte // Compressed BLS signature of the voted block hash (33 bytes)
	PubKeyBLS []byte // Sender BLS public key (129 bytes)
}

// NewBlockReduction returns a BlockReduction struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewBlockReduction(hash, sigBLS, pubKeyBLS []byte) (*BlockReduction, error) {
	if len(hash) != 32 {
		return nil, errors.New("wire: supplied candidate hash for reduction payload is improper length")
	}

	if len(sigBLS) != 33 {
		return nil, errors.New("wire: supplied compressed BLS signature for reduction payload is improper length")
	}

	return &BlockReduction{
		BlockHash: hash,
		SigBLS:    sigBLS,
		PubKeyBLS: pubKeyBLS,
	}, nil
}

// Encode a BlockReduction struct and write to w.
// Implements Msg interface.
func (rd *BlockReduction) Encode(w io.Writer) error {
	if err := encoding.Write256(w, rd.BlockHash); err != nil {
		return err
	}

	if err := encoding.WriteBLS(w, rd.SigBLS); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, rd.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Decode a BlockReduction from r.
// Implements Msg interface.
func (rd *BlockReduction) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &rd.BlockHash); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &rd.SigBLS); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &rd.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Type returns the consensus payload identifier.
// Implements Msg interface.
func (rd *BlockReduction) Type() ID {
	return BlockReductionID
}

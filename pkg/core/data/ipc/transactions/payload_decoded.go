// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactions

import (
	"bytes"
	"encoding/binary"

	"golang.org/x/crypto/blake2b"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

// const size_BLS_SCALAR = 32
// const size_CONTRACTID = 32
// const capacity_MESSAGE = 2
// const size_CIPHER = capacity_MESSAGE + 1
// const size_CIPHER_BYTES = size_CIPHER * size_BLS_SCALAR
// const size_NOTE = 137 + size_CIPHER_BYTES
// const size_STEALTH_ADDRESS = 64
// const size_G1AFFINE = 48
// const size_COMMITMENT = size_G1AFFINE
// const size_PROOF_EVAL = 24 * size_BLS_SCALAR
// const size_PROOF = 15*size_COMMITMENT + size_PROOF_EVAL

// TransactionPayloadDecoded carries data for an execute transaction (type 1).
type TransactionPayloadDecoded struct {
	Anchor     []byte     `json:"anchor"`
	Nullifiers [][]byte   `json:"nullifier"`
	Crossover  *Crossover `json:"crossover"`
	Notes      []*Note    `json:"notes"`
	Fee        *Fee       `json:"fee"`
	SpendProof []byte     `json:"spend_proof"`
	Call       *Call      `json:"call"`
}

// NewTransactionPayloadDecoded returns a new empty TransactionPayloadDecoded struct.
func NewTransactionPayloadDecoded() *TransactionPayloadDecoded {
	return &TransactionPayloadDecoded{
		Nullifiers: make([][]byte, 0),
		Notes:      make([]*Note, 0),
		Anchor:     make([]byte, 32),
		SpendProof: make([]byte, 1488),
		Fee:        NewFee(),
		Crossover:  nil,
		Call:       nil,
	}
}

// Hash the decoded payload in the same way as the transaction is hashed in `dusk-wallet-core`.
func (p *TransactionPayloadDecoded) Hash(txType TxType) ([]byte, error) {
	hash, err := blake2b.New(32, nil)
	if err != nil {
		return nil, err
	}

	for _, nullifier := range p.Nullifiers {
		if _, err := hash.Write(nullifier); err != nil {
			return nil, err
		}
	}

	for _, note := range p.Notes {
		var buf bytes.Buffer
		if err := MarshalNote(&buf, note); err != nil {
			return nil, err
		}

		if _, err := hash.Write(buf.Bytes()); err != nil {
			return nil, err
		}
	}

	if _, err := hash.Write(p.Anchor); err != nil {
		return nil, err
	}

	leb := make([]byte, 8)

	binary.LittleEndian.PutUint64(leb, p.Fee.GasLimit)

	if _, err := hash.Write(leb); err != nil {
		return nil, err
	}

	binary.LittleEndian.PutUint64(leb, p.Fee.GasPrice)

	if _, err := hash.Write(leb); err != nil {
		return nil, err
	}

	if _, err := hash.Write(p.Fee.StealthAddr); err != nil {
		return nil, err
	}

	if p.Crossover != nil {
		if _, err := hash.Write(p.Crossover.ValueCommitment); err != nil {
			return nil, err
		}

		if _, err := hash.Write(p.Crossover.Nonce); err != nil {
			return nil, err
		}

		if _, err := hash.Write(p.Crossover.EncryptedData); err != nil {
			return nil, err
		}
	}

	if p.Call != nil {
		if _, err := hash.Write(p.Call.ContractID); err != nil {
			return nil, err
		}

		if _, err := hash.Write(p.Call.CallData); err != nil {
			return nil, err
		}
	}

	hashBytes := hash.Sum(nil)
	hashBytes[31] &= 0xf // truncate in the same way as `rusk-abi`

	return hashBytes, nil
}

// UnmarshalTransactionPayloadDecoded reads a TransactionPayloadDecoded struct from a bytes.Buffer.
func UnmarshalTransactionPayloadDecoded(r *bytes.Buffer, f *TransactionPayloadDecoded, txType TxType) error {
	var lenInputs uint64
	if err := encoding.ReadUint64LE(r, &lenInputs); err != nil {
		return err
	}

	f.Nullifiers = make([][]byte, lenInputs)
	for i := range f.Nullifiers {
		f.Nullifiers[i] = make([]byte, 32)
		if err := encoding.Read256(r, f.Nullifiers[i]); err != nil {
			return err
		}
	}

	var lenNotes uint64
	if err := encoding.ReadUint64LE(r, &lenNotes); err != nil {
		return err
	}

	f.Notes = make([]*Note, lenNotes)
	for i := range f.Notes {
		f.Notes[i] = NewNote()
		if err := UnmarshalNote(r, f.Notes[i]); err != nil {
			return err
		}
	}

	if err := encoding.Read256(r, f.Anchor); err != nil {
		return err
	}

	if err := UnmarshalFee(r, f.Fee); err != nil {
		return err
	}

	if _, err := r.Read(f.SpendProof); err != nil {
		return err
	}

	var crossoverFlag uint64
	if err := encoding.ReadUint64LE(r, &crossoverFlag); err != nil {
		return err
	}

	if crossoverFlag > 0 {
		f.Crossover = NewCrossover()
		if err := UnmarshalCrossover(r, f.Crossover); err != nil {
			return err
		}
	}

	var callFlag uint64
	if err := encoding.ReadUint64LE(r, &callFlag); err != nil {
		return err
	}

	if callFlag > 0 {
		f.Call = NewCall()
		if err := UnmarshalCall(r, f.Call); err != nil {
			return err
		}
	}

	return nil
}

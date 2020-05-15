package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// Transaction according to the Phoenix model
type Transaction struct {
	Inputs  []*TransactionInput  `json:"inputs,omitempty"`
	Outputs []*TransactionOutput `json:"outputs"`
	Fee     *TransactionOutput   `json:"fee,omitempty"`
	Proof   []byte               `json:"proof,omitempty"`
	Data    []byte               `json:"data,omitempty"`
	hash    []byte
}

// Fees calculates the fees for this transaction
func (t *Transaction) Fees() uint64 {
	return t.Fee.Note.TransparentValue
}

// CalculateHash complies with merkletree.Payload interface
func (t *Transaction) CalculateHash() ([]byte, error) {
	b := new(bytes.Buffer)
	if err := MarshalTransaction(b, *t); err != nil {
		return nil, err
	}

	return hash.Sha3256(b.Bytes())
}

// Obfuscated returns whether this transaction is transparent or otherwise
func (t *Transaction) Obfuscated() bool {
	return t.Outputs[0].Note.NoteType == OBFUSCATED
}

// Type complies with ContractCall interface. Returns "Tx"
func (t *Transaction) Type() TxType {
	return Tx
}

// StandardTx returns the transaction itself. It complies with the ContractCall
// interface
func (t *Transaction) StandardTx() *Transaction {
	return t
}

// MTx copies from rusk.Transaction to transactions.Transaction
func MTx(r *rusk.Transaction, t *Transaction) error {
	r.Inputs = make([]*rusk.TransactionInput, len(t.Inputs))
	r.Outputs = make([]*rusk.TransactionOutput, len(t.Outputs))
	for i := range t.Inputs {
		r.Inputs[i] = new(rusk.TransactionInput)
		if err := MTxIn(r.Inputs[i], t.Inputs[i]); err != nil {
			return err
		}
	}
	for i := range t.Outputs {
		r.Outputs[i] = new(rusk.TransactionOutput)
		if err := MTxOut(r.Outputs[i], t.Outputs[i]); err != nil {
			return err
		}
	}
	r.Fee = new(rusk.TransactionOutput)
	if err := MTxOut(r.Fee, t.Fee); err != nil {
		return err
	}

	r.Proof = make([]byte, len(t.Proof))
	copy(r.Proof, t.Proof)
	r.Data = make([]byte, len(t.Data))
	copy(r.Data, t.Data)
	return nil
}

// UTx copies from rusk.Transaction to transactions.Transaction
func UTx(r *rusk.Transaction, t *Transaction) error {
	t.Inputs = make([]*TransactionInput, len(r.Inputs))
	t.Outputs = make([]*TransactionOutput, len(r.Outputs))
	for i := range r.Inputs {
		t.Inputs[i] = new(TransactionInput)
		if err := UTxIn(r.Inputs[i], t.Inputs[i]); err != nil {
			return err
		}
	}
	for i := range r.Outputs {
		t.Outputs[i] = new(TransactionOutput)
		if err := UTxOut(r.Outputs[i], t.Outputs[i]); err != nil {
			return err
		}
	}
	t.Fee = new(TransactionOutput)
	if err := UTxOut(r.Fee, t.Fee); err != nil {
		return err
	}

	t.Proof = make([]byte, len(r.Proof))
	copy(t.Proof, r.Proof)
	t.Data = make([]byte, len(r.Data))
	copy(t.Data, r.Data)
	return nil
}

//MarshalTransaction into a buffer
func MarshalTransaction(r *bytes.Buffer, t Transaction) error {
	if err := encoding.WriteVarInt(r, uint64(len(t.Inputs))); err != nil {
		return err
	}
	for _, input := range t.Inputs {
		if err := MarshalTransactionInput(r, *input); err != nil {
			return err
		}
	}

	if err := encoding.WriteVarInt(r, uint64(len(t.Outputs))); err != nil {
		return err
	}
	for _, output := range t.Outputs {
		if err := MarshalTransactionOutput(r, *output); err != nil {
			return err
		}
	}

	if err := MarshalTransactionOutput(r, *t.Fee); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, t.Proof); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, t.hash); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, t.Data); err != nil {
		return err
	}
	return nil
}

// UnmarshalTransaction from a buffer
func UnmarshalTransaction(r *bytes.Buffer, t *Transaction) error {
	nIn, eerr := encoding.ReadVarInt(r)
	if eerr != nil {
		return eerr
	}

	t.Inputs = make([]*TransactionInput, nIn)
	for i := range t.Inputs {
		tIn := new(TransactionInput)
		if err := UnmarshalTransactionInput(r, tIn); err != nil {
			return err
		}
		t.Inputs[i] = tIn
	}

	nOut, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	t.Outputs = make([]*TransactionOutput, nOut)
	for i := range t.Outputs {
		tOut := new(TransactionOutput)
		if err := UnmarshalTransactionOutput(r, tOut); err != nil {
			return err
		}
		t.Outputs[i] = tOut
	}

	t.Fee = new(TransactionOutput)
	if err := UnmarshalTransactionOutput(r, t.Fee); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &t.Proof); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &t.hash); err != nil {
		return err
	}
	if err := encoding.ReadVarBytes(r, &t.Data); err != nil {
		return err
	}
	return nil
}

// TransactionInput includes the notes, the nullifier and the transaction merkleroot
type TransactionInput struct {
	Nullifier  *Nullifier `json:"nullifier"`
	MerkleRoot *Scalar    `json:"merkle_root"`
}

// MTxIn copies from rusk.TransactionInput to transactions.TransactionInput
func MTxIn(r *rusk.TransactionInput, t *TransactionInput) error {
	r.Nullifier = new(rusk.Nullifier)
	r.MerkleRoot = new(rusk.Scalar)
	MNullifier(r.Nullifier, t.Nullifier)
	MScalar(r.MerkleRoot, t.MerkleRoot)
	return nil
}

// UTxIn copies from rusk.TransactionOutput to transactions.TransactionOutput
func UTxIn(r *rusk.TransactionInput, t *TransactionInput) error {
	t.Nullifier = new(Nullifier)
	t.MerkleRoot = new(Scalar)
	UNullifier(r.Nullifier, t.Nullifier)
	UScalar(r.MerkleRoot, t.MerkleRoot)
	return nil
}

// MarshalTransactionInput to a buffer
func MarshalTransactionInput(r *bytes.Buffer, t TransactionInput) error {
	if err := MarshalNullifier(r, *t.Nullifier); err != nil {
		return err
	}
	if err := MarshalScalar(r, *t.MerkleRoot); err != nil {
		return err
	}
	return nil
}

// UnmarshalTransactionInput from a buffer
func UnmarshalTransactionInput(r *bytes.Buffer, t *TransactionInput) error {
	t.Nullifier = new(Nullifier)
	if err := UnmarshalNullifier(r, t.Nullifier); err != nil {
		return err
	}
	t.MerkleRoot = new(Scalar)
	if err := UnmarshalScalar(r, t.MerkleRoot); err != nil {
		return err
	}
	return nil
}

// TransactionOutput is the spendable output of the transaction
type TransactionOutput struct {
	Note           *Note      `json:"note,omitempty"`
	Pk             *PublicKey `json:"pk,omitempty"`
	Value          uint64     `json:"value,omitempty"`
	BlindingFactor *Scalar    `json:"blinding_factor,omitempty"`
}

// MTxOut copies from transactions.TransactionOutput to rusk.TransactionOutput
func MTxOut(r *rusk.TransactionOutput, t *TransactionOutput) error {
	r.Note = new(rusk.Note)
	r.Pk = new(rusk.PublicKey)
	r.BlindingFactor = new(rusk.Scalar)

	if err := MNote(r.Note, t.Note); err != nil {
		return err
	}

	r.Value = t.Value
	MPublicKey(r.Pk, t.Pk)
	MScalar(r.BlindingFactor, t.BlindingFactor)
	return nil
}

// UTxOut copies from rusk.TransactionOutput to transactions.TransactionOutput
func UTxOut(r *rusk.TransactionOutput, t *TransactionOutput) error {
	t.Note = new(Note)
	t.Pk = new(PublicKey)
	t.BlindingFactor = new(Scalar)

	if err := UNote(r.Note, t.Note); err != nil {
		return err
	}

	t.Value = r.Value
	UPublicKey(r.Pk, t.Pk)
	UScalar(r.BlindingFactor, t.BlindingFactor)
	return nil
}

// MarshalTransactionOutput into a buffer
func MarshalTransactionOutput(r *bytes.Buffer, t TransactionOutput) error {
	if err := MarshalNote(r, *t.Note); err != nil {
		return err
	}
	if err := MarshalPublicKey(r, *t.Pk); err != nil {
		return err
	}
	if err := encoding.WriteUint64LE(r, t.Value); err != nil {
		return err
	}
	if err := MarshalScalar(r, *t.BlindingFactor); err != nil {
		return err
	}
	return nil
}

// UnmarshalTransactionOutput from a buffer
func UnmarshalTransactionOutput(r *bytes.Buffer, t *TransactionOutput) error {
	t.Note = &Note{}
	if err := UnmarshalNote(r, t.Note); err != nil {
		return err
	}
	t.Pk = &PublicKey{}
	if err := UnmarshalPublicKey(r, t.Pk); err != nil {
		return err
	}
	if err := encoding.ReadUint64LE(r, &t.Value); err != nil {
		return err
	}
	t.BlindingFactor = &Scalar{}
	if err := UnmarshalScalar(r, t.BlindingFactor); err != nil {
		return err
	}
	return nil
}

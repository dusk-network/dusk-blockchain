package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// TransactionPayload carries the common data contained in all transaction types.
type TransactionPayload struct {
	Anchor        *common.BlsScalar   `json:"anchor"`
	Nullifiers    []*common.BlsScalar `json:"nullifier"`
	*Crossover    `json:"crossover"`
	Notes         []*Note `json:"notes"`
	*Fee          `json:"fee"`
	SpendingProof *common.Proof `json:"spending_proof"`
	CallData      []byte        `json:"call_data"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (t *TransactionPayload) Copy() *TransactionPayload {
	inputs := make([]*common.BlsScalar, len(t.Nullifiers))
	for i := range inputs {
		inputs[i] = t.Nullifiers[i].Copy()
	}

	notes := make([]*Note, len(t.Notes))
	for i := range notes {
		notes[i] = t.Notes[i].Copy()
	}

	callData := make([]byte, len(t.CallData))
	copy(callData, t.CallData)

	return &TransactionPayload{
		Anchor:        t.Anchor.Copy(),
		Nullifiers:    inputs,
		Crossover:     t.Crossover.Copy(),
		Notes:         notes,
		Fee:           t.Fee.Copy(),
		SpendingProof: t.SpendingProof.Copy(),
		CallData:      callData,
	}
}

// MTransactionPayload copies the TransactionPayload structure into the Rusk equivalent.
func MTransactionPayload(r *rusk.TransactionPayload, f *TransactionPayload) {
	common.MBlsScalar(r.Anchor, f.Anchor)

	r.Nullifier = make([]*rusk.BlsScalar, len(f.Nullifiers))
	for i, input := range r.Nullifier {
		common.MBlsScalar(input, f.Nullifiers[i])
	}

	MCrossover(r.Crossover, f.Crossover)

	r.Notes = make([]*rusk.Note, len(f.Notes))
	for i, note := range r.Notes {
		MNote(note, f.Notes[i])
	}

	MFee(r.Fee, f.Fee)
	common.MProof(r.SpendingProof, f.SpendingProof)
	callData := make([]byte, len(f.CallData))
	copy(callData, f.CallData)
	r.CallData = callData
}

// UTransactionPayload copies the Rusk TransactionPayload structure into the native equivalent.
func UTransactionPayload(r *rusk.TransactionPayload, f *TransactionPayload) {
	common.UBlsScalar(r.Anchor, f.Anchor)

	f.Nullifiers = make([]*common.BlsScalar, len(r.Nullifier))
	for i, input := range f.Nullifiers {
		common.UBlsScalar(r.Nullifier[i], input)
	}

	UCrossover(r.Crossover, f.Crossover)

	f.Notes = make([]*Note, len(r.Notes))
	for i, note := range f.Notes {
		UNote(r.Notes[i], note)
	}

	UFee(r.Fee, f.Fee)
	common.UProof(r.SpendingProof, f.SpendingProof)
	callData := make([]byte, len(r.CallData))
	copy(callData, r.CallData)
	f.CallData = callData
}

// MarshalTransactionPayload writes the TransactionPayload struct into a bytes.Buffer.
func MarshalTransactionPayload(r *bytes.Buffer, f *TransactionPayload) error {
	if err := common.MarshalBlsScalar(r, f.Anchor); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(r, uint64(len(f.Nullifiers))); err != nil {
		return err
	}

	for _, input := range f.Nullifiers {
		if err := common.MarshalBlsScalar(r, input); err != nil {
			return err
		}
	}

	if err := MarshalCrossover(r, f.Crossover); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(r, uint64(len(f.Notes))); err != nil {
		return err
	}

	for _, note := range f.Notes {
		if err := MarshalNote(r, note); err != nil {
			return err
		}
	}

	if err := MarshalFee(r, f.Fee); err != nil {
		return err
	}

	if err := common.MarshalProof(r, f.SpendingProof); err != nil {
		return err
	}

	return encoding.WriteVarBytes(r, f.CallData)
}

// UnmarshalTransactionPayload reads a TransactionPayload struct from a bytes.Buffer.
func UnmarshalTransactionPayload(r *bytes.Buffer, f *TransactionPayload) error {
	f = new(TransactionPayload)

	if err := common.UnmarshalBlsScalar(r, f.Anchor); err != nil {
		return err
	}

	lenInputs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	f.Nullifiers = make([]*common.BlsScalar, lenInputs)
	for _, input := range f.Nullifiers {
		if err := common.UnmarshalBlsScalar(r, input); err != nil {
			return err
		}
	}

	if err := UnmarshalCrossover(r, f.Crossover); err != nil {
		return err
	}

	lenNotes, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	f.Notes = make([]*Note, lenNotes)
	for _, note := range f.Notes {
		if err := UnmarshalNote(r, note); err != nil {
			return err
		}
	}

	if err := UnmarshalFee(r, f.Fee); err != nil {
		return err
	}

	if err := common.UnmarshalProof(r, f.SpendingProof); err != nil {
		return err
	}

	return encoding.ReadVarBytes(r, &f.CallData)
}

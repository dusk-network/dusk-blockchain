package transactions

import (
	"bytes"
	"encoding/json"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// SlashTransaction is used by the consensus to slash the stake of Provisioners
// take vote multiple times within the same round
type SlashTransaction struct {
	*ContractTx
	Step      uint8  `json:"step"`
	Round     uint64 `json:"round"`
	BlsKey    []byte `json:"bls_key"`
	FirstMsg  []byte `json:"first_msg"`
	FirstSig  []byte `json:"first_sig"`
	SecondMsg []byte `json:"second_msg"`
	SecondSig []byte `json:"second_sig"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers
func (t *SlashTransaction) Copy() payload.Safe {
	cpy := &SlashTransaction{
		ContractTx: t.ContractTx.Copy(),
		Step:       t.Step,
		Round:      t.Round,
		BlsKey:     make([]byte, len(t.BlsKey)),
		FirstMsg:   make([]byte, len(t.FirstMsg)),
		FirstSig:   make([]byte, len(t.FirstSig)),
		SecondMsg:  make([]byte, len(t.SecondMsg)),
		SecondSig:  make([]byte, len(t.SecondSig)),
	}

	copy(cpy.BlsKey, t.BlsKey)
	copy(cpy.FirstMsg, t.FirstMsg)
	copy(cpy.FirstSig, t.FirstSig)
	copy(cpy.SecondMsg, t.SecondMsg)
	copy(cpy.SecondSig, t.SecondSig)

	return cpy
}

// MarshalJSON provides a json-encoded readable representation of a
// SlashTransaction
func (t *SlashTransaction) MarshalJSON() ([]byte, error) {
	// type aliasing allows to work around stack overflow of recursive JSON
	// marshaling
	type Alias SlashTransaction

	h, _ := t.CalculateHash()
	return json.Marshal(struct {
		*Alias
		jsonMarshalable
	}{
		Alias:           (*Alias)(t),
		jsonMarshalable: newJSONMarshalable(t.Type(), h),
	})
}

// MSlash copies the Slash transaction data to the rusk Slash
func MSlash(r *rusk.SlashTransaction, t *SlashTransaction) error {
	r.Tx = new(rusk.Transaction)
	if err := MTx(r.Tx, t.Tx); err != nil {
		return err
	}

	r.Step = uint32(t.Step)
	r.Round = t.Round
	r.BlsKey = make([]byte, len(t.BlsKey))
	copy(r.BlsKey, t.BlsKey)
	r.FirstMsg = make([]byte, len(t.FirstMsg))
	copy(r.FirstMsg, t.FirstMsg)
	r.FirstSig = make([]byte, len(t.FirstSig))
	copy(r.FirstSig, t.FirstSig)
	r.SecondMsg = make([]byte, len(t.SecondMsg))
	copy(r.SecondMsg, t.SecondMsg)
	r.SecondSig = make([]byte, len(t.SecondSig))
	copy(r.SecondSig, t.SecondSig)

	return nil
}

// USlash copies the Slash rusk struct into the transaction datastruct
func USlash(r *rusk.SlashTransaction, t *SlashTransaction) error {
	var err error
	t.ContractTx, err = UContractTx(r.Tx)
	if err != nil {
		return err
	}

	t.Step = uint8(r.Step)
	t.Round = r.Round
	t.BlsKey = make([]byte, len(r.BlsKey))
	copy(t.BlsKey, r.BlsKey)
	t.FirstMsg = make([]byte, len(r.FirstMsg))
	copy(t.FirstMsg, r.FirstMsg)
	t.FirstSig = make([]byte, len(r.FirstSig))
	copy(t.FirstSig, r.FirstSig)
	t.SecondMsg = make([]byte, len(r.SecondMsg))
	copy(t.SecondMsg, r.SecondMsg)
	t.SecondSig = make([]byte, len(r.SecondSig))
	copy(t.SecondSig, r.SecondSig)

	return nil
}

// CalculateHash complies with merkletree.Payload interface
func (t *SlashTransaction) CalculateHash() ([]byte, error) {
	b := new(bytes.Buffer)
	if err := MarshalSlash(b, *t); err != nil {
		return nil, err
	}

	return hash.Sha3256(b.Bytes())
}

//MarshalSlash into a buffer
func MarshalSlash(r *bytes.Buffer, s SlashTransaction) error {
	if err := MarshalContractTx(r, *s.ContractTx); err != nil {
		return err
	}

	if err := encoding.WriteUint8(r, s.Step); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, s.Round); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.BlsKey); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.FirstMsg); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.FirstSig); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.SecondMsg); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.SecondSig); err != nil {
		return err
	}

	return nil
}

//UnmarshalSlash into a buffer
func UnmarshalSlash(r *bytes.Buffer, s *SlashTransaction) error {
	s.ContractTx = new(ContractTx)

	if err := UnmarshalContractTx(r, s.ContractTx); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &s.Step); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &s.Round); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.BlsKey); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.FirstMsg); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.FirstSig); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.SecondMsg); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.SecondSig); err != nil {
		return err
	}

	return nil
}

// Type complies to the ContractCall interface
func (t *SlashTransaction) Type() TxType {
	return Slash
}

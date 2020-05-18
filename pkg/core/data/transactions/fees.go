package transactions

import (
	"bytes"
	"encoding/json"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// WithdrawFeesTransaction is one of the transactions to perform genesis
// contract calls. It is used by Provisioners to withdraw the fees they accrued
// with block validation
type WithdrawFeesTransaction struct {
	*ContractTx
	BlsKey []byte `json:"bls_key"`
	Sig    []byte `json:"sig"`
	Msg    []byte `json:"msg"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers
func (t *WithdrawFeesTransaction) Copy() payload.Safe {
	cpy := &WithdrawFeesTransaction{
		ContractTx: t.ContractTx.Copy(),
		BlsKey:     make([]byte, len(t.BlsKey)),
		Sig:        make([]byte, len(t.Sig)),
		Msg:        make([]byte, len(t.Msg)),
	}

	copy(cpy.BlsKey, t.BlsKey)
	copy(cpy.Sig, t.Sig)
	copy(cpy.Msg, t.Msg)

	return cpy
}

// MarshalJSON provides a json-encoded readable representation of a
// WithdrawFeesTransaction
func (t *WithdrawFeesTransaction) MarshalJSON() ([]byte, error) {
	// type aliasing allows to work around stack overflow of recursive JSON
	// marshaling
	type Alias WithdrawFeesTransaction

	h, _ := t.CalculateHash()
	return json.Marshal(struct {
		*Alias
		jsonMarshalable
	}{
		Alias:           (*Alias)(t),
		jsonMarshalable: newJSONMarshalable(t.Type(), h),
	})
}

// MWithdrawFees copies the WithdrawFeesTransaction into the rusk equivalent
func MWithdrawFees(r *rusk.WithdrawFeesTransaction, t *WithdrawFeesTransaction) error {
	r.Tx = new(rusk.Transaction)
	if err := MTx(r.Tx, t.Tx); err != nil {
		return err
	}
	r.BlsKey = make([]byte, len(t.BlsKey))
	copy(r.BlsKey, t.BlsKey)
	r.Sig = make([]byte, len(t.Sig))
	copy(r.Sig, t.Sig)
	r.Msg = make([]byte, len(t.Msg))
	copy(r.Msg, t.Msg)
	return nil
}

// UWithdrawFees copies the WithdrawFees rusk struct into the transaction datastruct
func UWithdrawFees(r *rusk.WithdrawFeesTransaction, t *WithdrawFeesTransaction) error {
	var err error
	t.ContractTx, err = UContractTx(r.Tx)
	if err != nil {
		return err
	}
	t.BlsKey = make([]byte, len(r.BlsKey))
	copy(t.BlsKey, r.BlsKey)
	t.Sig = make([]byte, len(r.Sig))
	copy(t.Sig, r.Sig)
	t.Msg = make([]byte, len(r.Msg))
	copy(t.Msg, r.Msg)
	return nil
}

// CalculateHash complies with merkletree.Payload interface
func (t *WithdrawFeesTransaction) CalculateHash() ([]byte, error) {
	b := new(bytes.Buffer)
	if err := MarshalFees(b, *t); err != nil {
		return nil, err
	}

	return hash.Sha3256(b.Bytes())
}

//MarshalFees into a buffer
func MarshalFees(r *bytes.Buffer, s WithdrawFeesTransaction) error {
	if err := MarshalContractTx(r, *s.ContractTx); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.BlsKey); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Sig); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Msg); err != nil {
		return err
	}

	return nil
}

//UnmarshalFees into a buffer
func UnmarshalFees(r *bytes.Buffer, s *WithdrawFeesTransaction) error {
	s.ContractTx = new(ContractTx)

	if err := UnmarshalContractTx(r, s.ContractTx); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.BlsKey); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.Sig); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.Msg); err != nil {
		return err
	}

	return nil
}

// Type complies to the ContractCall interface
func (t *WithdrawFeesTransaction) Type() TxType {
	return WithdrawFees
}

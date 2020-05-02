package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

// WithdrawFeesTransaction is one of the transactions to perform genesis
// contract calls. It is used by Provisioners to withdraw the fees they accrued
// with block validation
type WithdrawFeesTransaction struct {
	*ContractTx
	BlsKey []byte `protobuf:"bytes,2,opt,name=bls_key,json=blsKey,proto3" json:"bls_key,omitempty"`
	Sig    []byte `protobuf:"bytes,3,opt,name=sig,proto3" json:"sig,omitempty"`
	Msg    []byte `protobuf:"bytes,4,opt,name=msg,proto3" json:"msg,omitempty"`
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
func (w *WithdrawFeesTransaction) Type() TxType {
	return WithdrawFees
}

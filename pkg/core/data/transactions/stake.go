package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

// StakeTransaction is used by the nodes to create a stake and become
// Provisioners
type StakeTransaction struct {
	*ContractTx
	BlsKey           []byte `protobuf:"bytes,2,opt,name=bls_key,json=blsKey,proto3" json:"bls_key,omitempty"`
	Value            uint64 `protobuf:"fixed64,3,opt,name=value,proto3" json:"value,omitempty"` // Should be the leftover value that was burned in `tx`
	ExpirationHeight uint64 `protobuf:"fixed64,4,opt,name=expiration_height,json=expirationHeight,proto3" json:"expiration_height,omitempty"`
}

// Type complies with the ContractCall interface
func (s *StakeTransaction) Type() TxType {
	return Stake
}

//MarshalStake into a buffer
func MarshalStake(r *bytes.Buffer, s StakeTransaction) error {
	if err := MarshalContractTx(r, *s.ContractTx); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.BlsKey); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, s.Value); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, s.ExpirationHeight); err != nil {
		return err
	}

	return nil
}

//UnmarshalStake into a buffer
func UnmarshalStake(r *bytes.Buffer, s *StakeTransaction) error {
	s.ContractTx = new(ContractTx)

	if err := UnmarshalContractTx(r, s.ContractTx); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.BlsKey); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &s.Value); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &s.ExpirationHeight); err != nil {
		return err
	}

	return nil
}

// WithdrawStakeTransaction is the transaction used by Provisioners to withdraw
// the stakes
type WithdrawStakeTransaction struct {
	*ContractTx
	BlsKey []byte `protobuf:"bytes,2,opt,name=bls_key,json=blsKey,proto3" json:"bls_key,omitempty"`
	Sig    []byte `protobuf:"bytes,3,opt,name=sig,proto3" json:"sig,omitempty"`
}

//MarshalWithdrawStake into a buffer
func MarshalWithdrawStake(r *bytes.Buffer, s WithdrawStakeTransaction) error {
	if err := MarshalContractTx(r, *s.ContractTx); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.BlsKey); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Sig); err != nil {
		return err
	}

	if err := MarshalTransaction(r, *s.Tx); err != nil {
		return err
	}
	return nil
}

//UnmarshalWithdrawStake into a buffer
func UnmarshalWithdrawStake(r *bytes.Buffer, s *WithdrawStakeTransaction) error {
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

	return nil
}

// Type complies with the ContractCall interface
func (w *WithdrawStakeTransaction) Type() TxType {
	return WithdrawStake
}

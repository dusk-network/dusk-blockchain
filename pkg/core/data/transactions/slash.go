package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

// SlashTransaction is used by the consensus to slash the stake of Provisioners
// take vote multiple times within the same round
type SlashTransaction struct {
	*ContractTx
	BlsKey    []byte `protobuf:"bytes,2,opt,name=bls_key,json=blsKey,proto3" json:"bls_key,omitempty"`
	Step      uint32 `protobuf:"varint,3,opt,name=step,proto3" json:"step,omitempty"`
	Round     uint64 `protobuf:"fixed64,4,opt,name=round,proto3" json:"round,omitempty"`
	FirstMsg  []byte `protobuf:"bytes,5,opt,name=first_msg,json=firstMsg,proto3" json:"first_msg,omitempty"`
	FirstSig  []byte `protobuf:"bytes,6,opt,name=first_sig,json=firstSig,proto3" json:"first_sig,omitempty"`
	SecondMsg []byte `protobuf:"bytes,7,opt,name=second_msg,json=secondMsg,proto3" json:"second_msg,omitempty"`
	SecondSig []byte `protobuf:"bytes,8,opt,name=second_sig,json=secondSig,proto3" json:"second_sig,omitempty"`
}

//MarshalSlash into a buffer
func MarshalSlash(r *bytes.Buffer, s SlashTransaction) error {
	if err := MarshalContractTx(r, *s.ContractTx); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.BlsKey); err != nil {
		return err
	}

	step := uint8(s.Step)
	if err := encoding.WriteUint8(r, step); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, s.Round); err != nil {
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

	if err := encoding.ReadVarBytes(r, &s.BlsKey); err != nil {
		return err
	}

	var step uint8
	if err := encoding.ReadUint8(r, &step); err != nil {
		return err
	}
	s.Step = uint32(step)

	if err := encoding.ReadUint64LE(r, &s.Round); err != nil {
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
func (s *SlashTransaction) Type() TxType {
	return Slash
}

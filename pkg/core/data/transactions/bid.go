package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

// BidTransaction is the transaction created by BlockGenerators to be able to
// calculate the scores to accompany block candidates
type BidTransaction struct {
	*ContractTx
	M                []byte `protobuf:"bytes,2,opt,name=m,proto3" json:"m,omitempty"`
	Commitment       []byte `protobuf:"bytes,3,opt,name=commitment,proto3" json:"commitment,omitempty"`
	EncryptedValue   []byte `protobuf:"bytes,4,opt,name=encrypted_value,json=encryptedValue,proto3" json:"encrypted_value,omitempty"`
	EncryptedBlinder []byte `protobuf:"bytes,5,opt,name=encrypted_blinder,json=encryptedBlinder,proto3" json:"encrypted_blinder,omitempty"`
	ExpirationHeight uint64 `protobuf:"fixed64,6,opt,name=expiration_height,json=expirationHeight,proto3" json:"expiration_height,omitempty"`
	Pk               []byte `protobuf:"bytes,7,opt,name=pk,proto3" json:"pk,omitempty"`
	R                []byte `protobuf:"bytes,8,opt,name=r,proto3" json:"r,omitempty"`
	Seed             []byte `protobuf:"bytes,9,opt,name=seed,proto3" json:"seed,omitempty"`
}

//MarshalBid into a buffer
func MarshalBid(r *bytes.Buffer, s BidTransaction) error {
	if err := MarshalContractTx(r, *s.ContractTx); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.M); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Commitment); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.EncryptedValue); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.EncryptedBlinder); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, s.ExpirationHeight); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Pk); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.R); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Seed); err != nil {
		return err
	}

	return nil
}

//UnmarshalBid into a buffer
func UnmarshalBid(r *bytes.Buffer, s *BidTransaction) error {
	s.ContractTx = &ContractTx{}

	if err := UnmarshalContractTx(r, s.ContractTx); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.M); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.Commitment); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.EncryptedValue); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.EncryptedBlinder); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &s.ExpirationHeight); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.Pk); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.R); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.Seed); err != nil {
		return err
	}

	return nil
}

// Type complies with the ContractCall interface
func (b *BidTransaction) Type() TxType {
	return Bid
}

// WithdrawBidTransaction is the transaction used by BlockGenerators to
// withdraw their stake (bid)
type WithdrawBidTransaction struct {
	*ContractTx
	Commitment       []byte `protobuf:"bytes,2,opt,name=commitment,proto3" json:"commitment,omitempty"`
	EncryptedValue   []byte `protobuf:"bytes,3,opt,name=encrypted_value,json=encryptedValue,proto3" json:"encrypted_value,omitempty"`
	EncryptedBlinder []byte `protobuf:"bytes,4,opt,name=encrypted_blinder,json=encryptedBlinder,proto3" json:"encrypted_blinder,omitempty"`
	Bid              []byte `protobuf:"bytes,5,opt,name=bid,proto3" json:"bid,omitempty"`
	Sig              []byte `protobuf:"bytes,6,opt,name=sig,proto3" json:"sig,omitempty"`
}

//MarshalWithdrawBid into a buffer
func MarshalWithdrawBid(r *bytes.Buffer, s WithdrawBidTransaction) error {
	if err := MarshalContractTx(r, *s.ContractTx); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Commitment); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.EncryptedValue); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.EncryptedBlinder); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Bid); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Sig); err != nil {
		return err
	}

	return nil
}

//UnmarshalWithdrawBid into a buffer
func UnmarshalWithdrawBid(r *bytes.Buffer, s *WithdrawBidTransaction) error {
	s.ContractTx = &ContractTx{}

	if err := UnmarshalContractTx(r, s.ContractTx); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.Commitment); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.EncryptedValue); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.EncryptedBlinder); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.Bid); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.Sig); err != nil {
		return err
	}

	return nil
}

// Type complies with the ContractCall interface
func (w *WithdrawBidTransaction) Type() TxType {
	return WithdrawBid
}

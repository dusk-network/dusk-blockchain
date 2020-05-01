package transactions

// BidTransaction is the transaction created by BlockGenerators to be able to
// calculate the scores to accompany block candidates
type BidTransaction struct {
	*ContractTx
	M                []byte `protobuf:"bytes,1,opt,name=m,proto3" json:"m,omitempty"`
	Commitment       []byte `protobuf:"bytes,2,opt,name=commitment,proto3" json:"commitment,omitempty"`
	EncryptedValue   []byte `protobuf:"bytes,3,opt,name=encrypted_value,json=encryptedValue,proto3" json:"encrypted_value,omitempty"`
	EncryptedBlinder []byte `protobuf:"bytes,4,opt,name=encrypted_blinder,json=encryptedBlinder,proto3" json:"encrypted_blinder,omitempty"`
	ExpirationHeight uint64 `protobuf:"fixed64,5,opt,name=expiration_height,json=expirationHeight,proto3" json:"expiration_height,omitempty"`
	Pk               []byte `protobuf:"bytes,6,opt,name=pk,proto3" json:"pk,omitempty"`
	R                []byte `protobuf:"bytes,7,opt,name=r,proto3" json:"r,omitempty"`
	Seed             []byte `protobuf:"bytes,8,opt,name=seed,proto3" json:"seed,omitempty"`
}

// Type complies with the ContractCall interface
func (b *BidTransaction) Type() TxType {
	return Bid
}

// WithdrawBidTransaction is the transaction used by BlockGenerators to
// withdraw their stake (bid)
type WithdrawBidTransaction struct {
	*ContractTx
	Commitment       []byte       `protobuf:"bytes,1,opt,name=commitment,proto3" json:"commitment,omitempty"`
	EncryptedValue   []byte       `protobuf:"bytes,2,opt,name=encrypted_value,json=encryptedValue,proto3" json:"encrypted_value,omitempty"`
	EncryptedBlinder []byte       `protobuf:"bytes,3,opt,name=encrypted_blinder,json=encryptedBlinder,proto3" json:"encrypted_blinder,omitempty"`
	Bid              []byte       `protobuf:"bytes,4,opt,name=bid,proto3" json:"bid,omitempty"`
	Sig              []byte       `protobuf:"bytes,5,opt,name=sig,proto3" json:"sig,omitempty"`
	Tx               *Transaction `protobuf:"bytes,6,opt,name=tx,proto3" json:"tx,omitempty"`
}

// Type complies with the ContractCall interface
func (w *WithdrawBidTransaction) Type() TxType {
	return WithdrawBid
}

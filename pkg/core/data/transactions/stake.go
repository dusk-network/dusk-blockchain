package transactions

// StakeTransaction is used by the nodes to create a stake and become
// Provisioners
type StakeTransaction struct {
	*ContractTx
	BlsKey           []byte       `protobuf:"bytes,1,opt,name=bls_key,json=blsKey,proto3" json:"bls_key,omitempty"`
	Value            uint64       `protobuf:"fixed64,2,opt,name=value,proto3" json:"value,omitempty"` // Should be the leftover value that was burned in `tx`
	ExpirationHeight uint64       `protobuf:"fixed64,3,opt,name=expiration_height,json=expirationHeight,proto3" json:"expiration_height,omitempty"`
	Tx               *Transaction `protobuf:"bytes,4,opt,name=tx,proto3" json:"tx,omitempty"`
}

// Type complies with the ContractCall interface
func (s *StakeTransaction) Type() TxType {
	return Stake
}

// WithdrawStakeTransaction is the transaction used by Provisioners to withdraw
// the stakes
type WithdrawStakeTransaction struct {
	*ContractTx
	BlsKey []byte       `protobuf:"bytes,1,opt,name=bls_key,json=blsKey,proto3" json:"bls_key,omitempty"`
	Sig    []byte       `protobuf:"bytes,2,opt,name=sig,proto3" json:"sig,omitempty"`
	Tx     *Transaction `protobuf:"bytes,3,opt,name=tx,proto3" json:"tx,omitempty"`
}

// Type complies with the ContractCall interface
func (w *WithdrawStakeTransaction) Type() TxType {
	return WithdrawStake
}

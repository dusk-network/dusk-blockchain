package transactions

// WithdrawFeesTransaction is one of the transactions to perform genesis
// contract calls. It is used by Provisioners to withdraw the fees they accrued
// with block validation
type WithdrawFeesTransaction struct {
	*ContractTx
	BlsKey []byte       `protobuf:"bytes,1,opt,name=bls_key,json=blsKey,proto3" json:"bls_key,omitempty"`
	Sig    []byte       `protobuf:"bytes,2,opt,name=sig,proto3" json:"sig,omitempty"`
	Msg    []byte       `protobuf:"bytes,3,opt,name=msg,proto3" json:"msg,omitempty"`
	Tx     *Transaction `protobuf:"bytes,4,opt,name=tx,proto3" json:"tx,omitempty"`
}

// Type complies to the ContractCall interface
func (w *WithdrawFeesTransaction) Type() TxType {
	return WithdrawFees
}

package transactions

// SlashTransaction is used by the consensus to slash the stake of Provisioners
// take vote multiple times within the same round
type SlashTransaction struct {
	*ContractTx
	BlsKey    []byte       `protobuf:"bytes,1,opt,name=bls_key,json=blsKey,proto3" json:"bls_key,omitempty"`
	Step      uint32       `protobuf:"varint,2,opt,name=step,proto3" json:"step,omitempty"`
	Round     uint64       `protobuf:"fixed64,3,opt,name=round,proto3" json:"round,omitempty"`
	FirstMsg  []byte       `protobuf:"bytes,4,opt,name=first_msg,json=firstMsg,proto3" json:"first_msg,omitempty"`
	FirstSig  []byte       `protobuf:"bytes,5,opt,name=first_sig,json=firstSig,proto3" json:"first_sig,omitempty"`
	SecondMsg []byte       `protobuf:"bytes,6,opt,name=second_msg,json=secondMsg,proto3" json:"second_msg,omitempty"`
	SecondSig []byte       `protobuf:"bytes,7,opt,name=second_sig,json=secondSig,proto3" json:"second_sig,omitempty"`
	Tx        *Transaction `protobuf:"bytes,8,opt,name=tx,proto3" json:"tx,omitempty"`
}

// Type complies to the ContractCall interface
func (s *SlashTransaction) Type() TxType {
	return Slash
}

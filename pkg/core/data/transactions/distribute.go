package transactions

// DistributeTransaction includes coinbase output distribution and fee
// allocation to the Provisioners
type DistributeTransaction struct {
	*ContractTx
	TotalReward           uint64       `protobuf:"fixed64,1,opt,name=total_reward,json=totalReward,proto3" json:"total_reward,omitempty"`
	ProvisionersAddresses []byte       `protobuf:"bytes,2,opt,name=provisioners_addresses,json=provisionersAddresses,proto3" json:"provisioners_addresses,omitempty"`
	BgPk                  *PublicKey   `protobuf:"bytes,3,opt,name=bg_pk,json=bgPk,proto3" json:"bg_pk,omitempty"`
	Tx                    *Transaction `protobuf:"bytes,4,opt,name=tx,proto3" json:"tx,omitempty"`
}

// Type is part of the ContractCall interface
func (d *DistributeTransaction) Type() TxType {
	return Distribute
}

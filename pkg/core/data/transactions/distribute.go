package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

// DistributeTransaction includes coinbase output distribution and fee
// allocation to the Provisioners
type DistributeTransaction struct {
	*ContractTx
	TotalReward           uint64     `protobuf:"fixed64,2,opt,name=total_reward,json=totalReward,proto3" json:"total_reward,omitempty"`
	ProvisionersAddresses []byte     `protobuf:"bytes,3,opt,name=provisioners_addresses,json=provisionersAddresses,proto3" json:"provisioners_addresses,omitempty"`
	BgPk                  *PublicKey `protobuf:"bytes,4,opt,name=bg_pk,json=bgPk,proto3" json:"bg_pk,omitempty"`
}

//MarshalDistribute into a buffer
func MarshalDistribute(r *bytes.Buffer, s DistributeTransaction) error {
	if err := MarshalContractTx(r, *s.ContractTx); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, s.TotalReward); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.ProvisionersAddresses); err != nil {
		return err
	}

	if err := MarshalPublicKey(r, *s.BgPk); err != nil {
		return err
	}

	return nil
}

//UnmarshalDistribute into a buffer
func UnmarshalDistribute(r *bytes.Buffer, s *DistributeTransaction) error {
	s.ContractTx = &ContractTx{}

	if err := UnmarshalContractTx(r, s.ContractTx); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &s.TotalReward); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.ProvisionersAddresses); err != nil {
		return err
	}

	s.BgPk = &PublicKey{}
	if err := UnmarshalPublicKey(r, s.BgPk); err != nil {
		return err
	}

	return nil
}

// Type is part of the ContractCall interface
func (d *DistributeTransaction) Type() TxType {
	return Distribute
}

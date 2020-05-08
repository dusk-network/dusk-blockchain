package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// DistributeTransaction includes coinbase output distribution and fee
// allocation to the Provisioners
type DistributeTransaction struct {
	*ContractTx
	TotalReward           uint64     `protobuf:"fixed64,2,opt,name=total_reward,json=totalReward,proto3" json:"total_reward,omitempty"`
	ProvisionersAddresses []byte     `protobuf:"bytes,3,opt,name=provisioners_addresses,json=provisionersAddresses,proto3" json:"provisioners_addresses,omitempty"`
	BgPk                  *PublicKey `protobuf:"bytes,4,opt,name=bg_pk,json=bgPk,proto3" json:"bg_pk,omitempty"`
}

// CalculateHash complies with merkletree.Payload interface
func (t *DistributeTransaction) CalculateHash() ([]byte, error) {
	b := new(bytes.Buffer)
	if err := MarshalDistribute(b, *t); err != nil {
		return nil, err
	}

	return hash.Sha3256(b.Bytes())
}

// MDistribute copies the Distribute struct into the rusk  datastruct
func MDistribute(r *rusk.DistributeTransaction, t *DistributeTransaction) error {
	r.Tx = new(rusk.Transaction)
	if err := MTx(r.Tx, t.Tx); err != nil {
		return err
	}
	r.TotalReward = t.TotalReward
	r.ProvisionersAddresses = make([]byte, len(t.ProvisionersAddresses))
	copy(r.ProvisionersAddresses, t.ProvisionersAddresses)
	UPublicKey(r.BgPk, t.BgPk)
	return nil
}

// UDistribute copies the Distribute rusk struct into the transaction datastruct
func UDistribute(r *rusk.DistributeTransaction, t *DistributeTransaction) error {
	var err error
	t.ContractTx, err = UContractTx(r.Tx)
	if err != nil {
		return err
	}
	t.TotalReward = r.TotalReward
	t.ProvisionersAddresses = make([]byte, len(r.ProvisionersAddresses))
	copy(t.ProvisionersAddresses, r.ProvisionersAddresses)
	UPublicKey(r.BgPk, t.BgPk)
	return nil
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
	s.ContractTx = new(ContractTx)

	if err := UnmarshalContractTx(r, s.ContractTx); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &s.TotalReward); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.ProvisionersAddresses); err != nil {
		return err
	}

	s.BgPk = new(PublicKey)
	if err := UnmarshalPublicKey(r, s.BgPk); err != nil {
		return err
	}

	return nil
}

// Type is part of the ContractCall interface
func (t *DistributeTransaction) Type() TxType {
	return Distribute
}

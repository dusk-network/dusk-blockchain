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
	ProvisionersAddresses [][]byte   `json:"provisioners_addresses"`
	BgPk                  *PublicKey `json:"bg_pk"`
}

func newDistribute() *DistributeTransaction {
	dt := new(DistributeTransaction)
	dt.ContractTx = new(ContractTx)
	dt.ContractTx.Tx = new(Transaction)
	return dt
}

// TotalReward returns the reward set in the DistributeTransaction (coinbase).
// It is the transparent value of the first output
func (t *DistributeTransaction) TotalReward() uint64 {
	return t.Tx.Outputs[0].Note.TransparentValue
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
	r.ProvisionersAddresses = make([][]byte, len(t.ProvisionersAddresses))
	for i, p := range t.ProvisionersAddresses {
		r.ProvisionersAddresses[i] = make([]byte, len(p))
		copy(r.ProvisionersAddresses[i], p)
	}
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
	t.ProvisionersAddresses = make([][]byte, len(r.ProvisionersAddresses))
	for i, p := range r.ProvisionersAddresses {
		t.ProvisionersAddresses[i] = make([]byte, len(p))
		copy(t.ProvisionersAddresses[i], p)
	}
	UPublicKey(r.BgPk, t.BgPk)
	return nil
}

//MarshalDistribute into a buffer
func MarshalDistribute(r *bytes.Buffer, s DistributeTransaction) error {
	if err := MarshalContractTx(r, *s.ContractTx); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(r, uint64(len(s.ProvisionersAddresses))); err != nil {
		return err
	}

	for _, p := range s.ProvisionersAddresses {
		if err := encoding.WriteVarInt(r, uint64(len(p))); err != nil {
			return err
		}

		if err := encoding.WriteVarBytes(r, p); err != nil {
			return err
		}
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

	nrProvs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	s.ProvisionersAddresses = make([][]byte, nrProvs)
	for i := 0; i < int(nrProvs); i++ {
		blsKeySize, err := encoding.ReadVarInt(r)
		if err != nil {
			return err
		}

		s.ProvisionersAddresses[i] = make([]byte, blsKeySize)
		if err := encoding.ReadVarBytes(r, &s.ProvisionersAddresses[i]); err != nil {
			return err
		}
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

package transactions

import (
	"bytes"
	"encoding/json"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
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

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers
func (t *DistributeTransaction) Copy() payload.Safe {
	cpy := &DistributeTransaction{
		BgPk: t.BgPk.Copy(),
	}

	cpy.ProvisionersAddresses = make([][]byte, len(t.ProvisionersAddresses))
	for i, b := range t.ProvisionersAddresses {
		cpy.ProvisionersAddresses[i] = make([]byte, len(b))
		copy(cpy.ProvisionersAddresses[i], b)
	}
	return cpy
}

// MarshalJSON provides a json-encoded readable representation of a
// DistributeTransaction
func (t *DistributeTransaction) MarshalJSON() ([]byte, error) {

	type NullableTx struct {
		Outputs []*TransactionOutput `json:"outputs"`
	}

	h, _ := t.CalculateHash()
	btx := NullableTx{
		Outputs: []*TransactionOutput{t.Tx.Outputs[0]},
	}

	return json.Marshal(struct {
		jsonMarshalable
		ProvisionersAddresses [][]byte   `json:"provisioners_addresses"`
		BgPk                  *PublicKey `json:"bg_pk"`
		Tx                    NullableTx `json:"tx"`
	}{
		jsonMarshalable:       newJSONMarshalable(t.Type(), h),
		ProvisionersAddresses: t.ProvisionersAddresses,
		BgPk:                  t.BgPk,
		Tx:                    btx,
	})
}

// Values returns a tuple where the first element is the sum of all transparent
// outputs' note values and the second is the fee
func (t *DistributeTransaction) Values() (amount uint64, fee uint64) {
	amount = t.TotalReward()
	fee = 0
	return
}

// Obfuscated returns false for DistributeTransaction. We do not rely on the
// embedded ContractTx.Obfuscated method since it is unclear whether a
// (Transparent) Note will keep being available for coinbase transactions, as
// they are the only ones created outside of RUSK
func (t *DistributeTransaction) Obfuscated() bool {
	return false
}

// Fees is zero for coinbase (it is the only transaction that carries 0 fees)
func (t *DistributeTransaction) Fees() uint64 {
	return uint64(0)
}

// NewDistribute creates a DistributeTransaction. We can instantiate a
// DistributeTransaction directly because it only carries a uint64 value
// without the need for a Fee, Inputs, CompressedPoiints or Scalar quantities
func NewDistribute(reward uint64, provisioners [][]byte, bgPk PublicKey) *DistributeTransaction {
	dt := new(DistributeTransaction)
	dt.ContractTx = new(ContractTx)
	dt.ContractTx.Tx = new(Transaction)
	dt.ContractTx.Tx.Outputs = make([]*TransactionOutput, 1)
	dt.ContractTx.Tx.Outputs[0] = new(TransactionOutput)
	dt.ContractTx.Tx.Outputs[0].Note = new(Note)
	dt.ContractTx.Tx.Outputs[0].Note.TransparentValue = reward
	dt.ProvisionersAddresses = provisioners
	dt.BgPk = &bgPk
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
	r.Tx.Outputs = make([]*rusk.TransactionOutput, 1)
	r.Tx.Outputs[0] = new(rusk.TransactionOutput)
	r.Tx.Outputs[0].Note = new(rusk.Note)
	r.Tx.Outputs[0].Note.Value = &rusk.Note_TransparentValue{
		TransparentValue: t.TotalReward(),
	}
	r.ProvisionersAddresses = t.ProvisionersAddresses
	r.BgPk = new(rusk.PublicKey)
	MPublicKey(r.BgPk, t.BgPk)
	return nil
}

//MarshalDistribute into a buffer
func MarshalDistribute(r *bytes.Buffer, s DistributeTransaction) error {
	reward := s.Tx.Outputs[0].Note.TransparentValue
	if err := encoding.WriteUint64LE(r, reward); err != nil {
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
	s.ContractTx.Tx = new(Transaction)
	s.ContractTx.Tx.Outputs = make([]*TransactionOutput, 1)
	s.ContractTx.Tx.Outputs[0] = new(TransactionOutput)
	s.ContractTx.Tx.Outputs[0].Note = new(Note)

	if err := encoding.ReadUint64LE(r, &s.ContractTx.Tx.Outputs[0].Note.TransparentValue); err != nil {
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

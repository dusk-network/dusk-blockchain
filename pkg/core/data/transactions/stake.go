package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// StakeTransaction is used by the nodes to create a stake and become
// Provisioners
type StakeTransaction struct {
	*ContractTx
	BlsKey           []byte `json:"bls_key"`
	ExpirationHeight uint64 `json:"expiration_height"`
}

// Amount returns the amount staked through this transaction. The amount is the
// tranparent value of the note of the first of the transaction's outputs
func (t *StakeTransaction) Amount() uint64 {
	return t.Tx.Outputs[0].Note.TransparentValue
}

func newStake() *StakeTransaction {
	stake := new(StakeTransaction)
	stake.ContractTx = new(ContractTx)
	stake.ContractTx.Tx = new(Transaction)
	return stake
}

// MStake copies the Stake struct into the rusk datastruct
func MStake(r *rusk.StakeTransaction, t *StakeTransaction) error {
	r.Tx = new(rusk.Transaction)
	if err := MTx(r.Tx, t.Tx); err != nil {
		return err
	}
	r.BlsKey = make([]byte, len(t.BlsKey))
	copy(r.BlsKey, t.BlsKey)
	r.ExpirationHeight = t.ExpirationHeight
	return nil
}

// UStake copies the Stake rusk struct into the transaction datastruct
func UStake(r *rusk.StakeTransaction, t *StakeTransaction) error {
	var err error
	t.ContractTx, err = UContractTx(r.Tx)
	if err != nil {
		return err
	}
	t.BlsKey = make([]byte, len(r.BlsKey))
	copy(t.BlsKey, r.BlsKey)
	t.ExpirationHeight = r.ExpirationHeight
	return nil
}

// CalculateHash complies with merkletree.Payload interface
func (t *StakeTransaction) CalculateHash() ([]byte, error) {
	b := new(bytes.Buffer)
	if err := MarshalStake(b, *t); err != nil {
		return nil, err
	}

	return hash.Sha3256(b.Bytes())
}

// Type complies with the ContractCall interface
func (t *StakeTransaction) Type() TxType {
	return Stake
}

//MarshalStake into a buffer
func MarshalStake(r *bytes.Buffer, s StakeTransaction) error {
	if err := MarshalContractTx(r, *s.ContractTx); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.BlsKey); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, s.ExpirationHeight); err != nil {
		return err
	}

	return nil
}

//UnmarshalStake into a buffer
func UnmarshalStake(r *bytes.Buffer, s *StakeTransaction) error {
	s.ContractTx = new(ContractTx)

	if err := UnmarshalContractTx(r, s.ContractTx); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.BlsKey); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &s.ExpirationHeight); err != nil {
		return err
	}

	return nil
}

// WithdrawStakeTransaction is the transaction used by Provisioners to withdraw
// the stakes
type WithdrawStakeTransaction struct {
	*ContractTx
	BlsKey []byte `json:"bls_key"`
	Sig    []byte `json:"sig"`
}

// MWithdrawStake copies the WithdrawStake rusk struct into the transaction datastruct
func MWithdrawStake(r *rusk.WithdrawStakeTransaction, t *WithdrawStakeTransaction) error {
	r.Tx = new(rusk.Transaction)
	if err := MTx(r.Tx, t.Tx); err != nil {
		return err
	}
	r.BlsKey = make([]byte, len(t.BlsKey))
	copy(r.BlsKey, t.BlsKey)
	r.Sig = make([]byte, len(t.Sig))
	copy(r.Sig, t.Sig)
	return nil
}

// UWithdrawStake copies the WithdrawStake rusk struct into the transaction datastruct
func UWithdrawStake(r *rusk.WithdrawStakeTransaction, t *WithdrawStakeTransaction) error {
	var err error
	t.ContractTx, err = UContractTx(r.Tx)
	if err != nil {
		return err
	}
	t.BlsKey = make([]byte, len(r.BlsKey))
	copy(t.BlsKey, r.BlsKey)
	t.Sig = make([]byte, len(r.Sig))
	copy(t.Sig, r.Sig)
	return nil
}

// CalculateHash complies with merkletree.Payload interface
func (t *WithdrawStakeTransaction) CalculateHash() ([]byte, error) {
	b := new(bytes.Buffer)
	if err := MarshalWithdrawStake(b, *t); err != nil {
		return nil, err
	}

	return hash.Sha3256(b.Bytes())
}

//MarshalWithdrawStake into a buffer
func MarshalWithdrawStake(r *bytes.Buffer, s WithdrawStakeTransaction) error {
	if err := MarshalContractTx(r, *s.ContractTx); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.BlsKey); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Sig); err != nil {
		return err
	}

	if err := MarshalTransaction(r, *s.Tx); err != nil {
		return err
	}
	return nil
}

//UnmarshalWithdrawStake into a buffer
func UnmarshalWithdrawStake(r *bytes.Buffer, s *WithdrawStakeTransaction) error {
	s.ContractTx = new(ContractTx)

	if err := UnmarshalContractTx(r, s.ContractTx); err != nil {
		return err
	}
	if err := encoding.ReadVarBytes(r, &s.BlsKey); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.Sig); err != nil {
		return err
	}

	return nil
}

// Type complies with the ContractCall interface
func (t *WithdrawStakeTransaction) Type() TxType {
	return WithdrawStake
}

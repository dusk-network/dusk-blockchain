package transactions

import (
	"bytes"
	"encoding/json"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// BidTransaction is the transaction created by BlockGenerators to be able to
// calculate the scores to accompany block candidates
type BidTransaction struct {
	*ContractTx
	M                []byte `json:"m"`
	Commitment       []byte `json:"commitment"`
	Pk               []byte `json:"pk"`
	R                []byte `json:"r"`
	Seed             []byte `json:"seed"`
	ExpirationHeight uint64 `json:"expiration_height"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers
func (t *BidTransaction) Copy() payload.Safe {
	b := new(BidTransaction)
	b.ContractTx = t.ContractTx.Copy()
	b.M = make([]byte, len(t.M))
	copy(b.M, t.M)
	b.Commitment = make([]byte, len(t.Commitment))
	copy(b.Commitment, t.Commitment)
	b.Pk = make([]byte, len(t.Pk))
	copy(b.Pk, t.Pk)
	b.R = make([]byte, len(t.R))
	copy(b.R, t.R)
	b.Seed = make([]byte, len(t.Seed))
	copy(b.Seed, t.Seed)
	b.ExpirationHeight = t.ExpirationHeight
	return b
}

// MarshalJSON provides a json-encoded readable representation of a
// BidTransaction
func (t *BidTransaction) MarshalJSON() ([]byte, error) {
	// type aliasing allows to work around stack overflow of recursive JSON
	// marshaling
	type Alias BidTransaction

	h, _ := t.CalculateHash()
	return json.Marshal(struct {
		*Alias
		jsonMarshalable
	}{
		Alias:           (*Alias)(t),
		jsonMarshalable: newJSONMarshalable(t.Type(), h),
	})
}

func newBid() *BidTransaction {
	bt := new(BidTransaction)
	bt.ContractTx = new(ContractTx)
	bt.ContractTx.Tx = new(Transaction)
	return bt
}

// MBid copies the Bid rusk struct into the transaction datastruct
func MBid(r *rusk.BidTransaction, t *BidTransaction) error {
	r.Tx = new(rusk.Transaction)
	if err := MTx(r.Tx, t.Tx); err != nil {
		return err
	}

	r.M = make([]byte, len(t.M))
	copy(r.M, t.M)
	r.Commitment = make([]byte, len(t.Commitment))
	copy(r.Commitment, t.Commitment)
	r.Pk = make([]byte, len(t.Pk))
	copy(r.Pk, t.Pk)
	r.R = make([]byte, len(t.R))
	copy(r.R, t.R)
	r.Seed = make([]byte, len(t.Seed))
	copy(r.Seed, t.Seed)
	r.ExpirationHeight = t.ExpirationHeight
	return nil
}

// UBid copies the Bid rusk struct into the transaction datastruct
func UBid(r *rusk.BidTransaction, t *BidTransaction) error {
	var err error
	t.ContractTx, err = UContractTx(r.Tx)
	if err != nil {
		return err
	}

	t.M = make([]byte, len(r.M))
	copy(t.M, r.M)
	t.Commitment = make([]byte, len(r.Commitment))
	copy(t.Commitment, r.Commitment)
	t.Pk = make([]byte, len(r.Pk))
	copy(t.Pk, r.Pk)
	t.R = make([]byte, len(r.R))
	copy(t.R, r.R)
	t.Seed = make([]byte, len(r.Seed))
	copy(t.Seed, r.Seed)
	t.ExpirationHeight = r.ExpirationHeight
	return nil
}

// CalculateHash complies with merkletree.Payload interface
func (t *BidTransaction) CalculateHash() ([]byte, error) {
	b := new(bytes.Buffer)
	if err := MarshalBid(b, *t); err != nil {
		return nil, err
	}

	return hash.Sha3256(b.Bytes())
}

//MarshalBid into a buffer
func MarshalBid(r *bytes.Buffer, s BidTransaction) error {
	if err := MarshalContractTx(r, *s.ContractTx); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.M); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Commitment); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, s.ExpirationHeight); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Pk); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.R); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Seed); err != nil {
		return err
	}

	return nil
}

//UnmarshalBid into a buffer
func UnmarshalBid(r *bytes.Buffer, s *BidTransaction) error {
	s.ContractTx = new(ContractTx)
	if err := UnmarshalContractTx(r, s.ContractTx); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.M); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.Commitment); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &s.ExpirationHeight); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.Pk); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.R); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.Seed); err != nil {
		return err
	}

	return nil
}

// Type complies with the ContractCall interface
func (t *BidTransaction) Type() TxType {
	return Bid
}

// WithdrawBidTransaction is the transaction used by BlockGenerators to
// withdraw their stake (bid)
type WithdrawBidTransaction struct {
	*ContractTx
	Commitment []byte `json:"commitment"`
	Sig        []byte `json:"sig"`
	EdPk       []byte `json:"ed_pk"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers
func (t *WithdrawBidTransaction) Copy() payload.Safe {
	cpy := &WithdrawBidTransaction{
		ContractTx: t.ContractTx.Copy(),
		Commitment: make([]byte, len(t.Commitment)),
		Sig:        make([]byte, len(t.Sig)),
		EdPk:       make([]byte, len(t.EdPk)),
	}

	copy(cpy.Commitment, t.Commitment)
	copy(cpy.Sig, t.Sig)
	copy(cpy.EdPk, t.EdPk)

	return cpy
}

// MarshalJSON provides a json-encoded readable representation of a
// WithdrawBidTransaction
func (t *WithdrawBidTransaction) MarshalJSON() ([]byte, error) {
	// type aliasing allows to work around stack overflow of recursive JSON
	// marshaling
	type Alias WithdrawBidTransaction

	h, _ := t.CalculateHash()
	return json.Marshal(struct {
		*Alias
		jsonMarshalable
	}{
		Alias:           (*Alias)(t),
		jsonMarshalable: newJSONMarshalable(t.Type(), h),
	})
}

// CalculateHash complies with merkletree.Payload interface
func (t *WithdrawBidTransaction) CalculateHash() ([]byte, error) {
	b := new(bytes.Buffer)
	if err := MarshalWithdrawBid(b, *t); err != nil {
		return nil, err
	}

	return hash.Sha3256(b.Bytes())
}

// MWithdrawBid copies the WithdrawBid struct into the rusk datastruct
func MWithdrawBid(r *rusk.WithdrawBidTransaction, t *WithdrawBidTransaction) error {
	r.Tx = new(rusk.Transaction)
	if err := MTx(r.Tx, t.Tx); err != nil {
		return err
	}

	r.Commitment = make([]byte, len(t.Commitment))
	copy(r.Commitment, t.Commitment)
	r.Sig = make([]byte, len(t.Sig))
	copy(r.Sig, t.Sig)
	r.EdPk = make([]byte, len(t.EdPk))
	copy(r.EdPk, t.EdPk)
	return nil
}

// UWithdrawBid copies the WithdrawBid rusk struct into the transaction datastruct
func UWithdrawBid(r *rusk.WithdrawBidTransaction, t *WithdrawBidTransaction) error {
	var err error
	t.ContractTx, err = UContractTx(r.Tx)
	if err != nil {
		return err
	}
	t.Commitment = make([]byte, len(r.Commitment))
	copy(t.Commitment, r.Commitment)
	t.EdPk = make([]byte, len(r.EdPk))
	copy(t.EdPk, r.EdPk)
	t.Sig = make([]byte, len(r.Sig))
	copy(t.Sig, r.Sig)
	return nil
}

//MarshalWithdrawBid into a buffer
func MarshalWithdrawBid(r *bytes.Buffer, s WithdrawBidTransaction) error {
	if err := MarshalContractTx(r, *s.ContractTx); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Commitment); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.Sig); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, s.EdPk); err != nil {
		return err
	}

	return nil
}

//UnmarshalWithdrawBid into a buffer
func UnmarshalWithdrawBid(r *bytes.Buffer, s *WithdrawBidTransaction) error {
	s.ContractTx = &ContractTx{}

	if err := UnmarshalContractTx(r, s.ContractTx); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.Commitment); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.Sig); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.EdPk); err != nil {
		return err
	}

	return nil
}

// Type complies with the ContractCall interface
func (t *WithdrawBidTransaction) Type() TxType {
	return WithdrawBid
}

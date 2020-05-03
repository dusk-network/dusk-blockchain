package transactions

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-crypto/merkletree"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// TxType specifies the type of transaction, whether it is a contract call or a
// genesis contract transaction
type TxType uint8

const (
	// Tx indicates the phoenix transaction type
	Tx TxType = iota
	// Distribute indicates the coinbase and reward distribution contract call
	Distribute
	// WithdrawFees indicates the Provisioners' withdraw contract call
	WithdrawFees
	// Bid transaction propagated by the Block Generator
	Bid
	// Stake transaction propagated by the Provisioners
	Stake
	// Slash transaction propagated by the consensus to punish the Committee
	// members when they turn byzantine
	Slash
	// WithdrawStake transaction propagated by the Provisioners to withdraw
	// their stake
	WithdrawStake
	// WithdrawBid transaction propagated by the Block Generator to withdraw
	// their bids
	WithdrawBid
)

// ContractCall is the transaction that embodies the execution parameter for a
// smart contract method invocation
type ContractCall interface {
	merkletree.Payload
	// Type indicates the transaction
	Type() TxType
	// StandardTx is the underlying phoenix transaction carrying the
	// transaction outputs and inputs
	StandardTx() *Transaction
}

// ContractTx is the embedded struct utilized to group operations on the
// phoenix underlying transaction common to all genesis contracts
// It is NOT a ContractCall implementation as Transaction is a type apart
type ContractTx struct {
	Tx *Transaction `protobuf:"bytes,1,opt,name=tx,proto3" json:"tx,omitempty"`
}

// UContractTx is embedded by the ContractCall structs to facilitate copying
// the transaction fields from rusk.Transaction into Transaction data
func UContractTx(r *rusk.Transaction) (*ContractTx, error) {
	ctx := new(ContractTx)
	ctx.Tx = new(Transaction)
	if err := UTx(r, ctx.Tx); err != nil {
		return nil, err
	}
	return ctx, nil
}

// MarshalContractTx into a buffer
func MarshalContractTx(r *bytes.Buffer, c ContractTx) error {
	return MarshalTransaction(r, *c.Tx)
}

// UnmarshalContractTx from a buffer
func UnmarshalContractTx(r *bytes.Buffer, c *ContractTx) error {
	c.Tx = new(Transaction)
	return UnmarshalTransaction(r, c.Tx)
}

// StandardTx returns the underlying phoenix transaction
func (t *ContractTx) StandardTx() *Transaction {
	return t.Tx
}

// Equal tests two contract calls for equality
func Equal(a, b ContractCall) bool {
	ba, _ := a.CalculateHash()
	bb, _ := b.CalculateHash()
	return bytes.Equal(ba, bb)
}

// Marshal a ContractCall into a binary buffer
func Marshal(c ContractCall) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint8(buf, uint8(c.Type())); err != nil {
		return nil, err
	}

	var err error
	switch t := c.(type) {
	case *Transaction:
		err = MarshalTransaction(buf, *t)
	case *WithdrawFeesTransaction:
		err = MarshalFees(buf, *t)
	case *StakeTransaction:
		err = MarshalStake(buf, *t)
	case *BidTransaction:
		err = MarshalBid(buf, *t)
	case *SlashTransaction:
		err = MarshalSlash(buf, *t)
	case *DistributeTransaction:
		err = MarshalDistribute(buf, *t)
	case *WithdrawStakeTransaction:
		err = MarshalWithdrawStake(buf, *t)
	case *WithdrawBidTransaction:
		err = MarshalWithdrawBid(buf, *t)
	default:
		return nil, errors.New("structure to encode is not a contract call")
	}

	return buf, err
}

// Unmarshal the binary buffer into a ContractCall interface
func Unmarshal(buf *bytes.Buffer, c ContractCall) error {
	var txPrefix uint8
	if err := encoding.ReadUint8(buf, &txPrefix); err != nil {
		return err
	}

	var err error
	switch TxType(txPrefix) {
	case Tx:
		err = UnmarshalTransaction(buf, c.(*Transaction))
	case WithdrawFees:
		err = UnmarshalFees(buf, c.(*WithdrawFeesTransaction))
	case Stake:
		err = UnmarshalStake(buf, c.(*StakeTransaction))
	case Bid:
		err = UnmarshalBid(buf, c.(*BidTransaction))
	case Slash:
		err = UnmarshalSlash(buf, c.(*SlashTransaction))
	case Distribute:
		err = UnmarshalDistribute(buf, c.(*DistributeTransaction))
	case WithdrawStake:
		err = UnmarshalWithdrawStake(buf, c.(*WithdrawStakeTransaction))
	case WithdrawBid:
		err = UnmarshalWithdrawBid(buf, c.(*WithdrawBidTransaction))
	default:
		return errors.New("Unknown transaction type")
	}

	return err
}

// DecodeContractCall turns the protobuf message into a ContractCall
func DecodeContractCall(contractCall *rusk.ContractCallTx) (ContractCall, error) {
	switch c := contractCall.ContractCall.(type) {
	case *rusk.ContractCallTx_Tx:
		call := new(Transaction)
		if err := UTx(c.Tx, call); err != nil {
			return nil, err
		}
		return call, nil
	case *rusk.ContractCallTx_Withdraw:
		call := new(WithdrawFeesTransaction)
		if err := UWithdrawFees(c.Withdraw, call); err != nil {
			return nil, err
		}
		return call, nil
	case *rusk.ContractCallTx_Stake:
		call := new(StakeTransaction)
		if err := UStake(c.Stake, call); err != nil {
			return nil, err
		}
		return call, nil
	case *rusk.ContractCallTx_Bid:
		call := new(BidTransaction)
		if err := UBid(c.Bid, call); err != nil {
			return nil, err
		}
		return call, nil
	case *rusk.ContractCallTx_Slash:
		call := new(SlashTransaction)
		if err := USlash(c.Slash, call); err != nil {
			return nil, err
		}
		return call, nil
	case *rusk.ContractCallTx_Distribute:
		call := new(DistributeTransaction)
		if err := UDistribute(c.Distribute, call); err != nil {
			return nil, err
		}
		return call, nil
	case *rusk.ContractCallTx_WithdrawStake:
		call := new(WithdrawStakeTransaction)
		if err := UWithdrawStake(c.WithdrawStake, call); err != nil {
			return nil, err
		}
		return call, nil
	case *rusk.ContractCallTx_WithdrawBid:
		call := new(WithdrawBidTransaction)
		if err := UWithdrawBid(c.WithdrawBid, call); err != nil {
			return nil, err
		}
		return call, nil
	}

	return nil, errors.New("Unrecognized contract call")
}

// EncodeContractCall into a ContractCallTx
func EncodeContractCall(c interface{}) (*rusk.ContractCallTx, error) {
	switch t := c.(type) {
	case *Transaction:
		call := &rusk.ContractCallTx_Tx{
			Tx: new(rusk.Transaction),
		}
		if err := MTx(call.Tx, t); err != nil {
			return nil, err
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	case *WithdrawFeesTransaction:
		call := &rusk.ContractCallTx_Withdraw{
			Withdraw: new(rusk.WithdrawFeesTransaction),
		}
		if err := MWithdrawFees(call.Withdraw, t); err != nil {
			return nil, err
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	case *StakeTransaction:
		call := &rusk.ContractCallTx_Stake{
			Stake: new(rusk.StakeTransaction),
		}
		if err := MStake(call.Stake, t); err != nil {
			return nil, err
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	case *BidTransaction:
		call := &rusk.ContractCallTx_Bid{
			Bid: new(rusk.BidTransaction),
		}
		if err := MBid(call.Bid, t); err != nil {
			return nil, err
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	case *SlashTransaction:
		call := &rusk.ContractCallTx_Slash{
			Slash: new(rusk.SlashTransaction),
		}
		if err := MSlash(call.Slash, t); err != nil {
			return nil, err
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	case *DistributeTransaction:
		call := &rusk.ContractCallTx_Distribute{
			Distribute: new(rusk.DistributeTransaction),
		}
		if err := MDistribute(call.Distribute, t); err != nil {
			return nil, err
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	case *WithdrawStakeTransaction:
		call := &rusk.ContractCallTx_WithdrawStake{
			WithdrawStake: new(rusk.WithdrawStakeTransaction),
		}
		if err := MWithdrawStake(call.WithdrawStake, t); err != nil {
			return nil, err
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	case *WithdrawBidTransaction:
		call := &rusk.ContractCallTx_WithdrawBid{
			WithdrawBid: new(rusk.WithdrawBidTransaction),
		}
		if err := MWithdrawBid(call.WithdrawBid, t); err != nil {
			return nil, err
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	default:
		return nil, errors.New("structure to encode is not a contract call")
	}
}

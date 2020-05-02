package transactions

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-crypto/hash"
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

	// setHash is used to set a precalculated hash
	setHash([]byte)
}

// ContractTx is the embedded struct utilized to group operations on the
// phoenix underlying transaction common to all genesis contracts
type ContractTx struct {
	Tx *Transaction `protobuf:"bytes,1,opt,name=tx,proto3" json:"tx,omitempty"`
}

// MarshalContractTx into a buffer
func MarshalContractTx(r *bytes.Buffer, c ContractTx) error {
	return MarshalTransaction(r, *c.Tx)
}

// UnmarshalContractTx from a buffer
func UnmarshalContractTx(r *bytes.Buffer, c *ContractTx) error {
	c.Tx = &Transaction{}
	return UnmarshalTransaction(r, c.Tx)
}

// StandardTx returns the underlying phoenix transaction
func (t *ContractTx) StandardTx() *Transaction {
	return t.Tx
}

// CalculateHash returns the hash precalculated during serialization
func (t *ContractTx) CalculateHash() ([]byte, error) {
	return t.Tx.CalculateHash()
}

func (t *ContractTx) setHash(h []byte) {
	t.Tx.setHash(h)
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
	var call ContractCall

	var byteArr, h []byte
	var err error

	switch c := contractCall.ContractCall.(type) {
	case *rusk.ContractCallTx_Tx:
		byteArr, err = json.Marshal(c.Tx)
		if err != nil {
			return nil, err
		}
		call = new(Transaction)
	case *rusk.ContractCallTx_Withdraw:
		byteArr, err = json.Marshal(c.Withdraw)
		if err != nil {
			return nil, err
		}
		call = new(WithdrawFeesTransaction)
	case *rusk.ContractCallTx_Stake:
		byteArr, err = json.Marshal(c.Stake)
		if err != nil {
			return nil, err
		}
		call = new(StakeTransaction)
	case *rusk.ContractCallTx_Bid:
		byteArr, err = json.Marshal(c.Bid)
		if err != nil {
			return nil, err
		}
		call = new(BidTransaction)
	case *rusk.ContractCallTx_Slash:
		byteArr, err = json.Marshal(c.Slash)
		if err != nil {
			return nil, err
		}
		call = new(SlashTransaction)
	case *rusk.ContractCallTx_Distribute:
		byteArr, err = json.Marshal(c.Distribute)
		if err != nil {
			return nil, err
		}
		call = new(DistributeTransaction)
	case *rusk.ContractCallTx_WithdrawStake:
		byteArr, err = json.Marshal(c.WithdrawStake)
		if err != nil {
			return nil, err
		}
		call = new(WithdrawStakeTransaction)
	case *rusk.ContractCallTx_WithdrawBid:
		byteArr, err = json.Marshal(c.WithdrawBid)
		if err != nil {
			return nil, err
		}
		call = new(WithdrawBidTransaction)
	}

	h, err = hash.Sha3256(byteArr)
	if err != nil {
		return nil, err
	}

	call.setHash(h)

	if err := json.Unmarshal(byteArr, call); err != nil {
		return nil, err
	}

	return call, nil
}

// EncodeContractCall into a ContractCallTx
func EncodeContractCall(c interface{}) (*rusk.ContractCallTx, error) {
	var byteArr []byte
	var err error

	switch t := c.(type) {
	case *Transaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}

		call := &rusk.ContractCallTx_Tx{
			Tx: new(rusk.Transaction),
		}

		if jerr := json.Unmarshal(byteArr, call.Tx); jerr != nil {
			return nil, jerr
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	case *WithdrawFeesTransaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}

		call := &rusk.ContractCallTx_Withdraw{
			Withdraw: new(rusk.WithdrawFeesTransaction),
		}

		if jerr := json.Unmarshal(byteArr, call.Withdraw); jerr != nil {
			return nil, jerr
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	case *StakeTransaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}

		call := &rusk.ContractCallTx_Stake{
			Stake: new(rusk.StakeTransaction),
		}

		if jerr := json.Unmarshal(byteArr, call.Stake); jerr != nil {
			return nil, jerr
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	case *BidTransaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}

		call := &rusk.ContractCallTx_Bid{
			Bid: new(rusk.BidTransaction),
		}

		if jerr := json.Unmarshal(byteArr, call.Bid); jerr != nil {
			return nil, jerr
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	case *SlashTransaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}

		call := &rusk.ContractCallTx_Slash{
			Slash: new(rusk.SlashTransaction),
		}

		if jerr := json.Unmarshal(byteArr, call.Slash); jerr != nil {
			return nil, jerr
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	case *DistributeTransaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}

		call := &rusk.ContractCallTx_Distribute{
			Distribute: new(rusk.DistributeTransaction),
		}

		if jerr := json.Unmarshal(byteArr, call.Distribute); jerr != nil {
			return nil, jerr
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	case *WithdrawStakeTransaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}

		call := &rusk.ContractCallTx_WithdrawStake{
			WithdrawStake: new(rusk.WithdrawStakeTransaction),
		}

		if jerr := json.Unmarshal(byteArr, call.WithdrawStake); jerr != nil {
			return nil, jerr
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	case *WithdrawBidTransaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}

		call := &rusk.ContractCallTx_WithdrawBid{
			WithdrawBid: new(rusk.WithdrawBidTransaction),
		}

		if jerr := json.Unmarshal(byteArr, call.WithdrawBid); jerr != nil {
			return nil, jerr
		}
		return &rusk.ContractCallTx{ContractCall: call}, nil
	default:
		return nil, errors.New("structure to encode is not a contract call")
	}
}

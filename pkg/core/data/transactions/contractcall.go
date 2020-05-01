package transactions

import (
	"bytes"
	"encoding/json"
	"errors"

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

// NoteType is either Transparent or Obfuscated
type NoteType int32

const (
	// NoteType_TRANSPARENT denotes a public note (transaction output)
	NoteType_TRANSPARENT NoteType = 0
	// NoteType_OBFUSCATED denotes a private note (transaction output)
	NoteType_OBFUSCATED NoteType = 1
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
	Tx *Transaction `protobuf:"bytes,9,opt,name=tx,proto3" json:"tx,omitempty"`
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

// Transaction according to the Phoenix model
type Transaction struct {
	Inputs  []*TransactionInput  `protobuf:"bytes,1,rep,name=inputs,proto3" json:"inputs,omitempty"`
	Outputs []*TransactionOutput `protobuf:"bytes,2,rep,name=outputs,proto3" json:"outputs,omitempty"`
	Fee     *TransactionOutput   `protobuf:"bytes,3,opt,name=fee,proto3" json:"fee,omitempty"`
	Proof   []byte               `protobuf:"bytes,4,opt,name=proof,proto3" json:"proof,omitempty"`
	hash    []byte
}

func (t *Transaction) setHash(h []byte) {
	t.hash = h
}

// CalculateHash complies with merkletree.Payload interface
func (t *Transaction) CalculateHash() ([]byte, error) {
	return t.hash, nil
}

// Type complies with the ContractCall interface
func (t *Transaction) Type() TxType {
	return Tx
}

// StandardTx complies with the ContractCall interface. It returns the underlying
// phoenix transaction
func (t *Transaction) StandardTx() *Transaction {
	return t
}

// TransactionInput includes the notes, the nullifier and the transaction merkleroot
type TransactionInput struct {
	Note       *Note      `protobuf:"bytes,1,opt,name=note,proto3" json:"note,omitempty"`
	Pos        uint64     `protobuf:"fixed64,2,opt,name=pos,proto3" json:"pos,omitempty"`
	Sk         *SecretKey `protobuf:"bytes,3,opt,name=sk,proto3" json:"sk,omitempty"`
	Nullifier  *Nullifier `protobuf:"bytes,4,opt,name=nullifier,proto3" json:"nullifier,omitempty"`
	MerkleRoot *Scalar    `protobuf:"bytes,5,opt,name=merkle_root,json=merkleRoot,proto3" json:"merkle_root,omitempty"`
}

// TransactionOutput is the spendable output of the transaction
type TransactionOutput struct {
	Note           *Note      `protobuf:"bytes,1,opt,name=note,proto3" json:"note,omitempty"`
	Pk             *PublicKey `protobuf:"bytes,2,opt,name=pk,proto3" json:"pk,omitempty"`
	Value          uint64     `protobuf:"fixed64,3,opt,name=value,proto3" json:"value,omitempty"`
	BlindingFactor *Scalar    `protobuf:"bytes,4,opt,name=blinding_factor,json=blindingFactor,proto3" json:"blinding_factor,omitempty"`
}

// Nullifier of the transaction
type Nullifier struct {
	H *Scalar `protobuf:"bytes,1,opt,name=h,proto3" json:"h,omitempty"`
}

// Note is the spendable output
type Note struct {
	NoteType                  NoteType         `protobuf:"varint,1,opt,name=note_type,json=noteType,proto3,enum=rusk.NoteType" json:"note_type,omitempty"`
	Pos                       uint64           `protobuf:"fixed64,2,opt,name=pos,proto3" json:"pos,omitempty"`
	Nonce                     *Nonce           `protobuf:"bytes,3,opt,name=nonce,proto3" json:"nonce,omitempty"`
	RG                        *CompressedPoint `protobuf:"bytes,4,opt,name=r_g,json=rG,proto3" json:"r_g,omitempty"`
	PkR                       *CompressedPoint `protobuf:"bytes,5,opt,name=pk_r,json=pkR,proto3" json:"pk_r,omitempty"`
	ValueCommitment           *Scalar          `protobuf:"bytes,6,opt,name=value_commitment,json=valueCommitment,proto3" json:"value_commitment,omitempty"`
	TransparentBlindingFactor *Scalar          `protobuf:"bytes,7,opt,name=transparent_blinding_factor,json=transparentBlindingFactor,proto3,oneof"`
	EncryptedBlindingFactor   []byte           `protobuf:"bytes,8,opt,name=encrypted_blinding_factor,json=encryptedBlindingFactor,proto3,oneof"`
	TransparentValue          uint64           `protobuf:"fixed64,9,opt,name=transparent_value,json=transparentValue,proto3,oneof"`
	EncryptedValue            []byte           `protobuf:"bytes,10,opt,name=encrypted_value,json=encryptedValue,proto3,oneof"`
}

// SecretKey to sign the ContractCall
type SecretKey struct {
	A *Scalar `protobuf:"bytes,1,opt,name=a,proto3" json:"a,omitempty"`
	B *Scalar `protobuf:"bytes,2,opt,name=b,proto3" json:"b,omitempty"`
}

// ViewKey is to view the transactions belonging to the related SecretKey
type ViewKey struct {
	A  *Scalar          `protobuf:"bytes,1,opt,name=a,proto3" json:"a,omitempty"`
	BG *CompressedPoint `protobuf:"bytes,2,opt,name=b_g,json=bG,proto3" json:"b_g,omitempty"`
}

// PublicKey is the public key
type PublicKey struct {
	AG *CompressedPoint `protobuf:"bytes,1,opt,name=a_g,json=aG,proto3" json:"a_g,omitempty"`
	BG *CompressedPoint `protobuf:"bytes,2,opt,name=b_g,json=bG,proto3" json:"b_g,omitempty"`
}

// Scalar of the BLS12_381 curve
type Scalar struct {
	Data []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

// CompressedPoint of the BLS12_#81 curve
type CompressedPoint struct {
	Y []byte `protobuf:"bytes,1,opt,name=y,proto3" json:"y,omitempty"`
}

// Nonce is the distributed atomic increment used to prevent double spending
type Nonce struct {
	Bs []byte `protobuf:"bytes,1,opt,name=bs,proto3" json:"bs,omitempty"`
}

// Marshal a ContractCall into a binary buffer
// TODO marshal the hash
// TODO this should be the wire protocol
func Marshal(c ContractCall) (*bytes.Buffer, error) {
	var byteArr []byte
	var err error

	switch t := c.(type) {
	case *Transaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}
	case *WithdrawFeesTransaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}
	case *StakeTransaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}
	case *BidTransaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}
	case *SlashTransaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}
	case *DistributeTransaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}
	case *WithdrawStakeTransaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}
	case *WithdrawBidTransaction:
		byteArr, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("structure to encode is not a contract call")
	}

	return bytes.NewBuffer(byteArr), err
}

// Unmarshal the binary buffer into a ContractCall interface
// TODO unmarshal the hash
// TODO this should be the wire protocol
func Unmarshal(buf *bytes.Buffer, c ContractCall) error {
	return json.Unmarshal(buf.Bytes(), &c)
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

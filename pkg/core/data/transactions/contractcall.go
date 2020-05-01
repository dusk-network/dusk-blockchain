package transactions

import (
	"encoding/json"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

type NoteType int32

const (
	NoteType_TRANSPARENT NoteType = 0
	NoteType_OBFUSCATED  NoteType = 1
)

type DistributeTransaction struct {
	TotalReward           uint64       `protobuf:"fixed64,1,opt,name=total_reward,json=totalReward,proto3" json:"total_reward,omitempty"`
	ProvisionersAddresses []byte       `protobuf:"bytes,2,opt,name=provisioners_addresses,json=provisionersAddresses,proto3" json:"provisioners_addresses,omitempty"`
	BgPk                  *PublicKey   `protobuf:"bytes,3,opt,name=bg_pk,json=bgPk,proto3" json:"bg_pk,omitempty"`
	Tx                    *Transaction `protobuf:"bytes,4,opt,name=tx,proto3" json:"tx,omitempty"`
}

type WithdrawFeesTransaction struct {
	BlsKey []byte       `protobuf:"bytes,1,opt,name=bls_key,json=blsKey,proto3" json:"bls_key,omitempty"`
	Sig    []byte       `protobuf:"bytes,2,opt,name=sig,proto3" json:"sig,omitempty"`
	Msg    []byte       `protobuf:"bytes,3,opt,name=msg,proto3" json:"msg,omitempty"`
	Tx     *Transaction `protobuf:"bytes,4,opt,name=tx,proto3" json:"tx,omitempty"`
}

type SlashTransaction struct {
	BlsKey    []byte       `protobuf:"bytes,1,opt,name=bls_key,json=blsKey,proto3" json:"bls_key,omitempty"`
	Step      uint32       `protobuf:"varint,2,opt,name=step,proto3" json:"step,omitempty"`
	Round     uint64       `protobuf:"fixed64,3,opt,name=round,proto3" json:"round,omitempty"`
	FirstMsg  []byte       `protobuf:"bytes,4,opt,name=first_msg,json=firstMsg,proto3" json:"first_msg,omitempty"`
	FirstSig  []byte       `protobuf:"bytes,5,opt,name=first_sig,json=firstSig,proto3" json:"first_sig,omitempty"`
	SecondMsg []byte       `protobuf:"bytes,6,opt,name=second_msg,json=secondMsg,proto3" json:"second_msg,omitempty"`
	SecondSig []byte       `protobuf:"bytes,7,opt,name=second_sig,json=secondSig,proto3" json:"second_sig,omitempty"`
	Tx        *Transaction `protobuf:"bytes,8,opt,name=tx,proto3" json:"tx,omitempty"`
}

type StakeTransaction struct {
	BlsKey           []byte       `protobuf:"bytes,1,opt,name=bls_key,json=blsKey,proto3" json:"bls_key,omitempty"`
	Value            uint64       `protobuf:"fixed64,2,opt,name=value,proto3" json:"value,omitempty"` // Should be the leftover value that was burned in `tx`
	ExpirationHeight uint64       `protobuf:"fixed64,3,opt,name=expiration_height,json=expirationHeight,proto3" json:"expiration_height,omitempty"`
	Tx               *Transaction `protobuf:"bytes,4,opt,name=tx,proto3" json:"tx,omitempty"`
}

type BidTransaction struct {
	M                []byte       `protobuf:"bytes,1,opt,name=m,proto3" json:"m,omitempty"`
	Commitment       []byte       `protobuf:"bytes,2,opt,name=commitment,proto3" json:"commitment,omitempty"`
	EncryptedValue   []byte       `protobuf:"bytes,3,opt,name=encrypted_value,json=encryptedValue,proto3" json:"encrypted_value,omitempty"`
	EncryptedBlinder []byte       `protobuf:"bytes,4,opt,name=encrypted_blinder,json=encryptedBlinder,proto3" json:"encrypted_blinder,omitempty"`
	ExpirationHeight uint64       `protobuf:"fixed64,5,opt,name=expiration_height,json=expirationHeight,proto3" json:"expiration_height,omitempty"`
	Pk               []byte       `protobuf:"bytes,6,opt,name=pk,proto3" json:"pk,omitempty"`
	R                []byte       `protobuf:"bytes,7,opt,name=r,proto3" json:"r,omitempty"`
	Seed             []byte       `protobuf:"bytes,8,opt,name=seed,proto3" json:"seed,omitempty"`
	Tx               *Transaction `protobuf:"bytes,9,opt,name=tx,proto3" json:"tx,omitempty"`
}

type WithdrawBidTransaction struct {
	Commitment       []byte       `protobuf:"bytes,1,opt,name=commitment,proto3" json:"commitment,omitempty"`
	EncryptedValue   []byte       `protobuf:"bytes,2,opt,name=encrypted_value,json=encryptedValue,proto3" json:"encrypted_value,omitempty"`
	EncryptedBlinder []byte       `protobuf:"bytes,3,opt,name=encrypted_blinder,json=encryptedBlinder,proto3" json:"encrypted_blinder,omitempty"`
	Bid              []byte       `protobuf:"bytes,4,opt,name=bid,proto3" json:"bid,omitempty"`
	Sig              []byte       `protobuf:"bytes,5,opt,name=sig,proto3" json:"sig,omitempty"`
	Tx               *Transaction `protobuf:"bytes,6,opt,name=tx,proto3" json:"tx,omitempty"`
}

type WithdrawStakeTransaction struct {
	BlsKey []byte       `protobuf:"bytes,1,opt,name=bls_key,json=blsKey,proto3" json:"bls_key,omitempty"`
	Sig    []byte       `protobuf:"bytes,2,opt,name=sig,proto3" json:"sig,omitempty"`
	Tx     *Transaction `protobuf:"bytes,3,opt,name=tx,proto3" json:"tx,omitempty"`
}

type Transaction struct {
	Inputs  []*TransactionInput  `protobuf:"bytes,1,rep,name=inputs,proto3" json:"inputs,omitempty"`
	Outputs []*TransactionOutput `protobuf:"bytes,2,rep,name=outputs,proto3" json:"outputs,omitempty"`
	Fee     *TransactionOutput   `protobuf:"bytes,3,opt,name=fee,proto3" json:"fee,omitempty"`
	Proof   []byte               `protobuf:"bytes,4,opt,name=proof,proto3" json:"proof,omitempty"`
}

type TransactionInput struct {
	Note       *Note      `protobuf:"bytes,1,opt,name=note,proto3" json:"note,omitempty"`
	Pos        uint64     `protobuf:"fixed64,2,opt,name=pos,proto3" json:"pos,omitempty"`
	Sk         *SecretKey `protobuf:"bytes,3,opt,name=sk,proto3" json:"sk,omitempty"`
	Nullifier  *Nullifier `protobuf:"bytes,4,opt,name=nullifier,proto3" json:"nullifier,omitempty"`
	MerkleRoot *Scalar    `protobuf:"bytes,5,opt,name=merkle_root,json=merkleRoot,proto3" json:"merkle_root,omitempty"`
}

type TransactionOutput struct {
	Note           *Note      `protobuf:"bytes,1,opt,name=note,proto3" json:"note,omitempty"`
	Pk             *PublicKey `protobuf:"bytes,2,opt,name=pk,proto3" json:"pk,omitempty"`
	Value          uint64     `protobuf:"fixed64,3,opt,name=value,proto3" json:"value,omitempty"`
	BlindingFactor *Scalar    `protobuf:"bytes,4,opt,name=blinding_factor,json=blindingFactor,proto3" json:"blinding_factor,omitempty"`
}

type Nullifier struct {
	H *Scalar `protobuf:"bytes,1,opt,name=h,proto3" json:"h,omitempty"`
}

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

type Scalar struct {
	Data []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

type CompressedPoint struct {
	Y []byte `protobuf:"bytes,1,opt,name=y,proto3" json:"y,omitempty"`
}

type Nonce struct {
	Bs []byte `protobuf:"bytes,1,opt,name=bs,proto3" json:"bs,omitempty"`
}

type SecretKey struct {
	A *Scalar `protobuf:"bytes,1,opt,name=a,proto3" json:"a,omitempty"`
	B *Scalar `protobuf:"bytes,2,opt,name=b,proto3" json:"b,omitempty"`
}

type ViewKey struct {
	A  *Scalar          `protobuf:"bytes,1,opt,name=a,proto3" json:"a,omitempty"`
	BG *CompressedPoint `protobuf:"bytes,2,opt,name=b_g,json=bG,proto3" json:"b_g,omitempty"`
}

type PublicKey struct {
	AG *CompressedPoint `protobuf:"bytes,1,opt,name=a_g,json=aG,proto3" json:"a_g,omitempty"`
	BG *CompressedPoint `protobuf:"bytes,2,opt,name=b_g,json=bG,proto3" json:"b_g,omitempty"`
}

func DecodeContractCall(contractCall *rusk.ContractCallTx) (interface{}, error) {
	var call interface{}

	var byteArr []byte
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

	if err := json.Unmarshal(byteArr, call); err != nil {
		return nil, err
	}

	return call, nil
}

/*
type ContractCall struct {
	Inputs   []Input
	Outputs  []Output
	Fee      Output
	Proof    []byte
	CallData []byte
}

type Nullifier struct {
	H []byte
}

type Note struct {
	NoteType        uint8
	Pos             uint64
	Nonce           []byte
	RG              []byte
	PkR             []byte
	ValueCommitment []byte
	BlindingFactor  []byte
	Value           []byte
}

type Input struct {
	Nullifier
	MerkleRoot []byte
}

type Output struct {
	Note
	Recipient      []byte
	Value          uint64
	BlindingFactor []byte
}


func DecodeTx(tx *rusk.ContractCallTx_Tx) ContractCall {
	call := ContractCall{
		Inputs:  make([]Input, 0),
		Outputs: make([]Output, 0),
	}

	for _, input := range tx.Tx.Inputs {
		call.Inputs = append(call.Inputs, DecodeInput(input))
	}

	for _, output := range tx.Tx.Outputs {
		call.Outputs = append(call.Outputs, DecodeOutput(output)
	}

	call.Fee = DecodeOutput(tx.Tx.Fee)
	call.Proof = tx.Tx.Proof

	return call
}

func DecodeOutput(output *rusk.TransactionOutput) Output {
	out := Output{}

	out.Note = DecodeNote(output.Note)
	out.Recipient = append(output.PublicKey.AG, output.PublicKey.BG...)
	out.Value = output.Value
	out.BlindingFactor = output.BlindingFactor.Data
	return out
}

func DecodeInput(input *rusk.TransactionInput) Input {
	in := Input{}

	in.Nullifier.H = input.Nullifier.H
}

func DecodeNote(note *rusk.Note) Note {
	n := Note{}

}
*/

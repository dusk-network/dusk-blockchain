// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package legacy

import (
	"bytes"
	"math/big"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	newblock "github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	newtx "github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-crypto/mlsag"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/dusk-network/dusk-wallet/v2/block"
	"github.com/dusk-network/dusk-wallet/v2/transactions"

	log "github.com/sirupsen/logrus"
)

// NewBlockToOldBlock will convert a dusk-blockchain block into a dusk-wallet block.
func NewBlockToOldBlock(b *newblock.Block) (*block.Block, error) {
	ob := block.NewBlock()
	ob.Header = newHeaderToOldHeader(b.Header)
	rtxs := make([]*rusk.Transaction, len(b.Txs))

	for i, call := range b.Txs {
		tx := new(rusk.Transaction)
		if err := newtx.MTransaction(tx, call.(*newtx.Transaction)); err != nil {
			return nil, err
		}

		rtxs[i] = tx
	}

	txs, err := ContractCallsToTxs(rtxs)
	if err != nil {
		return nil, err
	}

	ob.Txs = txs
	return ob, nil
}

// OldBlockToNewBlock will convert a dusk-wallet block into a dusk-blockchain block.
func OldBlockToNewBlock(b *block.Block) (*newblock.Block, error) {
	nb := newblock.NewBlock()
	nb.Header = oldHeaderToNewHeader(b.Header)

	calls, err := txsToContractCalls(b.Txs)
	if err != nil {
		return nil, err
	}

	nb.Txs = calls
	return nb, nil
}

func newHeaderToOldHeader(h *newblock.Header) *block.Header {
	oh := block.NewHeader()
	oh.Version = h.Version
	oh.Height = h.Height
	oh.Timestamp = h.Timestamp
	oh.PrevBlockHash = h.PrevBlockHash
	oh.Seed = h.Seed
	oh.TxRoot = h.TxRoot
	oh.Certificate = newCertificateToOldCertificate(h.Certificate)
	oh.Hash = h.Hash
	return oh
}

func oldHeaderToNewHeader(h *block.Header) *newblock.Header {
	nh := newblock.NewHeader()
	nh.Version = h.Version
	nh.Height = h.Height
	nh.Timestamp = h.Timestamp
	nh.PrevBlockHash = h.PrevBlockHash
	nh.Seed = h.Seed
	nh.TxRoot = h.TxRoot
	nh.Certificate = oldCertificateToNewCertificate(h.Certificate)
	nh.Hash = h.Hash
	return nh
}

func newCertificateToOldCertificate(c *newblock.Certificate) *block.Certificate {
	oc := block.EmptyCertificate()
	oc.StepOneBatchedSig = c.StepOneBatchedSig
	oc.StepTwoBatchedSig = c.StepTwoBatchedSig
	oc.Step = c.Step
	oc.StepOneCommittee = c.StepOneCommittee
	oc.StepTwoCommittee = c.StepTwoCommittee
	return oc
}

func oldCertificateToNewCertificate(c *block.Certificate) *newblock.Certificate {
	nc := newblock.EmptyCertificate()
	nc.StepOneBatchedSig = c.StepOneBatchedSig
	nc.StepTwoBatchedSig = c.StepTwoBatchedSig
	nc.Step = c.Step
	nc.StepOneCommittee = c.StepOneCommittee
	nc.StepTwoCommittee = c.StepTwoCommittee
	return nc
}

// ProvisionersToRuskCommittee converts a native Provisioners struct to a slice of
// rusk Provisioners.
func ProvisionersToRuskCommittee(p *user.Provisioners) []*rusk.Provisioner {
	ruskProvisioners := make([]*rusk.Provisioner, len(p.Members))

	i := 0

	for _, n := range p.Members {
		ruskProvisioners[i] = new(rusk.Provisioner)
		ruskProvisioners[i].PublicKeyBls = n.PublicKeyBLS
		ruskProvisioners[i].Stakes = make([]*rusk.Stake, len(n.Stakes))

		for j, s := range n.Stakes {
			ruskProvisioners[i].Stakes[j] = new(rusk.Stake)
			ruskProvisioners[i].Stakes[j].Amount = s.Amount
			ruskProvisioners[i].Stakes[j].EndHeight = s.EndHeight
			ruskProvisioners[i].Stakes[j].StartHeight = s.StartHeight
		}

		i++
	}

	return ruskProvisioners
}

func txsToContractCalls(txs []transactions.Transaction) ([]newtx.ContractCall, error) {
	calls := make([]newtx.ContractCall, len(txs))

	for i, c := range txs {
		switch c.Type() {
		case transactions.CoinbaseType:
			fallthrough
		case transactions.StandardType:
			call, err := TxToRuskTx(c)
			if err != nil {
				return nil, err
			}

			// Replace the default type value (StandardType 0x03) with
			// Transfer Type (0x00)
			call.Type = 0

			tx := newtx.NewTransaction()
			if err := newtx.UTransaction(call, tx); err != nil {
				return nil, err
			}

			calls[i] = tx
		case transactions.StakeType:
			call, err := StakeToRuskStake(c.(*transactions.Stake))
			if err != nil {
				return nil, err
			}

			tx := newtx.NewTransaction()
			if err := newtx.UTransaction(call, tx); err != nil {
				return nil, err
			}

			calls[i] = tx
		case transactions.BidType:
			call, err := BidToRuskBid(c.(*transactions.Bid))
			if err != nil {
				return nil, err
			}

			tx := newtx.NewTransaction()
			if err := newtx.UTransaction(call.Tx, tx); err != nil {
				return nil, err
			}

			calls[i] = tx
		}
	}

	return calls, nil
}

// ContractCallsToTxs turns a slice of rusk contract calls into a slice of standard txs.
func ContractCallsToTxs(calls []*rusk.Transaction) ([]transactions.Transaction, error) {
	txs := make([]transactions.Transaction, len(calls))

	for i, call := range calls {
		switch call.Type {
		// Transfer
		case 0:
			tx, err := RuskTxToTx(call)
			if err != nil {
				log.WithError(err).
					WithField("type", call.Type).
					WithField("index", i).
					Errorln("error RuskTxToTx")
				return nil, err
			}

			txs[i] = tx
		// Distribute
		case 1:
			tx, err := RuskDistributeToCoinbase(call)
			if err != nil {
				log.WithError(err).
					WithField("type", call.Type).
					WithField("index", i).
					Errorln("error RuskTxToTx")
				return nil, err
			}

			txs[i] = tx
		// Bid
		case 3:
			tx, err := RuskBidToBid(call)
			if err != nil {
				log.WithError(err).
					WithField("type", call.Type).
					WithField("index", i).
					Errorln("error RuskTxToTx")
				return nil, err
			}

			txs[i] = tx
		// Stake
		case 4:
			tx, err := RuskStakeToStake(call)
			if err != nil {
				log.WithError(err).
					WithField("type", call.Type).
					WithField("index", i).
					Errorln("error RuskTxToTx")
				return nil, err
			}

			txs[i] = tx
		}
	}

	return txs, nil
}

func txToRuskTx(tx transactions.Transaction) (*newtx.Transaction, error) {
	buf := new(bytes.Buffer)

	if tx.Type() != transactions.CoinbaseType {
		if err := tx.StandardTx().RangeProof.Encode(buf, true); err != nil {
			return nil, err
		}
	}

	inputs, err := inputsToRuskInputs(tx.StandardTx().Inputs)
	if err != nil {
		return nil, err
	}

	outputs := outputsToRuskOutputs(tx.StandardTx().Outputs)

	pl := &newtx.TransactionPayload{
		Anchor:        tx.StandardTx().R.Bytes(),
		Nullifiers:    inputs,
		Notes:         outputs,
		Crossover:     newtx.MockCrossover(false),
		Fee:           newtx.MockFee(false),
		SpendingProof: buf.Bytes(),
		CallData:      make([]byte, 0),
	}

	return &newtx.Transaction{
		Version: 0,
		TxType:  newtx.TxType(0),
		Payload: pl,
	}, nil
}

// TxToRuskTx turns a legacy transaction into a rusk transaction.
func TxToRuskTx(tx transactions.Transaction) (*rusk.Transaction, error) {
	ntx, err := txToRuskTx(tx)
	if err != nil {
		return nil, err
	}

	plBuf := new(bytes.Buffer)
	if err := newtx.MarshalTransactionPayload(plBuf, ntx.Payload); err != nil {
		return nil, err
	}

	return &rusk.Transaction{
		Version: 0,
		Type:    uint32(tx.Type()),
		Payload: plBuf.Bytes(),
	}, nil
}

// RuskTxToTx turns a rusk transaction into a legacy transaction.
func RuskTxToTx(tx *rusk.Transaction) (*transactions.Standard, error) {
	var feeScalar ristretto.Scalar
	var rPoint ristretto.Point
	var dataArr [32]byte

	ntx := newtx.NewTransaction()
	if err := newtx.UTransaction(tx, ntx); err != nil {
		return nil, err
	}

	copy(dataArr[:], ntx.Payload.Anchor[0:32])

	feeScalar.SetBigInt(big.NewInt(int64(ntx.Payload.Fee.GasLimit)))
	rPoint.SetBytes(&dataArr)

	inputs, err := ruskInputsToInputs(ntx.Payload.Nullifiers)
	if err != nil {
		return nil, err
	}

	outputs := ruskOutputsToOutputs(ntx.Payload.Notes)
	stx := &transactions.Standard{
		TxType:  transactions.StandardType,
		R:       rPoint,
		Inputs:  inputs,
		Outputs: outputs,
		Version: 0,
		Fee:     feeScalar,
	}

	if len(ntx.Payload.SpendingProof) != 0 {
		if err := stx.RangeProof.Decode(bytes.NewBuffer(ntx.Payload.SpendingProof), true); err != nil {
			return nil, err
		}
	}

	return stx, nil
}

// StakeToRuskStake turns a legacy stake into a rusk stake.
func StakeToRuskStake(tx *transactions.Stake) (*rusk.Transaction, error) {
	rtx, err := txToRuskTx(tx.StandardTx())
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, tx.Lock); err != nil {
		return nil, err
	}

	if err := encoding.WriteVarBytes(buf, tx.PubKeyBLS); err != nil {
		return nil, err
	}

	rtx.Payload.CallData = buf.Bytes()

	plBuf := new(bytes.Buffer)
	if err := newtx.MarshalTransactionPayload(plBuf, rtx.Payload); err != nil {
		return nil, err
	}

	return &rusk.Transaction{
		Version: 0,
		Type:    4,
		Payload: plBuf.Bytes(),
	}, nil
}

// RuskStakeToStake turns a rusk stake into a legacy stake.
func RuskStakeToStake(tx *rusk.Transaction) (*transactions.Stake, error) {
	stx, err := RuskTxToTx(tx)
	if err != nil {
		return nil, err
	}

	stx.TxType = transactions.StakeType
	plBuf := bytes.NewBuffer(tx.Payload)

	pl := newtx.NewTransactionPayload()
	if err := newtx.UnmarshalTransactionPayload(plBuf, pl); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(pl.CallData)

	var expirationHeight uint64
	if err := encoding.ReadUint64LE(buf, &expirationHeight); err != nil {
		return nil, err
	}

	blsKey := make([]byte, 0)
	if err := encoding.ReadVarBytes(buf, &blsKey); err != nil {
		return nil, err
	}

	return &transactions.Stake{
		Timelock: &transactions.Timelock{
			Standard: stx,
			Lock:     expirationHeight,
		},
		PubKeyBLS: blsKey,
	}, nil
}

// BidToRuskBid turns a legacy bid into a rusk bid.
func BidToRuskBid(tx *transactions.Bid) (*rusk.BidTransaction, error) {
	rtx, err := txToRuskTx(tx.StandardTx())
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, tx.Lock); err != nil {
		return nil, err
	}

	if err := encoding.Write256(buf, tx.M); err != nil {
		return nil, err
	}

	rtx.Payload.CallData = buf.Bytes()

	plBuf := new(bytes.Buffer)
	if err := newtx.MarshalTransactionPayload(plBuf, rtx.Payload); err != nil {
		return nil, err
	}

	return &rusk.BidTransaction{
		BidTreeStorageIndex: 0,
		Tx: &rusk.Transaction{
			Version: 0,
			Type:    3,
			Payload: plBuf.Bytes(),
		},
	}, nil
}

// RuskBidToBid turns a rusk bid into a legacy bid.
func RuskBidToBid(tx *rusk.Transaction) (*transactions.Bid, error) {
	stx, err := RuskTxToTx(tx)
	if err != nil {
		return nil, err
	}

	stx.TxType = transactions.BidType
	plBuf := bytes.NewBuffer(tx.Payload)

	pl := newtx.NewTransactionPayload()
	if err := newtx.UnmarshalTransactionPayload(plBuf, pl); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(pl.CallData)

	var expirationHeight uint64
	if err := encoding.ReadUint64LE(buf, &expirationHeight); err != nil {
		return nil, err
	}

	M := make([]byte, 32)
	if err := encoding.Read256(buf, M); err != nil {
		return nil, err
	}

	return &transactions.Bid{
		Timelock: &transactions.Timelock{
			Standard: stx,
			Lock:     expirationHeight,
		},
		M: M,
	}, nil
}

// RuskDistributeToCoinbase turns a rusk distribute call to an equivalent legacy coinbase.
func RuskDistributeToCoinbase(tx *rusk.Transaction) (*transactions.Coinbase, error) {
	c := transactions.NewCoinbase(make([]byte, 10), make([]byte, 10), byte(2))
	plBuf := bytes.NewBuffer(tx.Payload)

	pl := newtx.NewTransactionPayload()
	if err := newtx.UnmarshalTransactionPayload(plBuf, pl); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(pl.CallData)

	var amount uint64
	if err := encoding.ReadUint64LE(buf, &amount); err != nil {
		return nil, err
	}

	var amountScalar ristretto.Scalar
	var pk ristretto.Scalar

	amountScalar.SetBigInt(new(big.Int).SetUint64(amount))
	pk.SetBigInt(new(big.Int).SetBytes(pl.Notes[0].PkR))

	c.Rewards = append(c.Rewards, &transactions.Output{
		EncryptedAmount: amountScalar,
		EncryptedMask:   pk,
	})

	return c, nil
}

func inputsToRuskInputs(inputs transactions.Inputs) ([][]byte, error) {
	rInputs := make([][]byte, len(inputs))

	for i, input := range inputs {
		buf := new(bytes.Buffer)
		if err := encoding.Write256(buf, input.KeyImage.Bytes()); err != nil {
			return nil, err
		}

		if err := encoding.Write256(buf, input.PubKey.P.Bytes()); err != nil {
			return nil, err
		}

		if err := encoding.Write256(buf, input.PseudoCommitment.Bytes()); err != nil {
			return nil, err
		}

		if err := input.Signature.Encode(buf, true); err != nil {
			return nil, err
		}

		rInputs[i] = buf.Bytes()
	}

	return rInputs, nil
}

func ruskInputsToInputs(inputs [][]byte) (transactions.Inputs, error) {
	sInputs := make(transactions.Inputs, len(inputs))

	for i, input := range inputs {
		sInputs[i] = new(transactions.Input)
		buf := bytes.NewBuffer(input)

		keyImageBytes := make([]byte, 32)
		if err := encoding.Read256(buf, keyImageBytes); err != nil {
			return nil, err
		}

		_ = sInputs[i].KeyImage.UnmarshalBinary(keyImageBytes)

		pubKeyBytes := make([]byte, 32)
		if err := encoding.Read256(buf, pubKeyBytes); err != nil {
			return nil, err
		}

		_ = sInputs[i].PubKey.P.UnmarshalBinary(pubKeyBytes)

		pseudoCommBytes := make([]byte, 32)
		if err := encoding.Read256(buf, pseudoCommBytes); err != nil {
			return nil, err
		}

		_ = sInputs[i].PseudoCommitment.UnmarshalBinary(pseudoCommBytes)

		sInputs[i].Signature = &mlsag.Signature{}
		if err := sInputs[i].Signature.Decode(buf, true); err != nil {
			return nil, err
		}
	}

	return sInputs, nil
}

func outputsToRuskOutputs(outputs transactions.Outputs) []*newtx.Note {
	rOutputs := make([]*newtx.Note, len(outputs))

	for i, output := range outputs {
		rOutputs[i] = &newtx.Note{
			Randomness:    output.Commitment.Bytes(),
			PkR:           output.PubKey.P.Bytes(),
			Commitment:    output.EncryptedAmount.Bytes(),
			Nonce:         output.EncryptedMask.Bytes(),
			EncryptedData: make([]byte, 96),
		}
	}

	return rOutputs
}

func ruskOutputsToOutputs(outputs []*newtx.Note) transactions.Outputs {
	sOutputs := make(transactions.Outputs, len(outputs))

	for i, output := range outputs {
		sOutputs[i] = new(transactions.Output)
		_ = sOutputs[i].Commitment.UnmarshalBinary(output.Randomness)
		_ = sOutputs[i].PubKey.P.UnmarshalBinary(output.PkR)
		_ = sOutputs[i].EncryptedAmount.UnmarshalBinary(output.Commitment)
		_ = sOutputs[i].EncryptedMask.UnmarshalBinary(output.Nonce)
	}

	return sOutputs
}

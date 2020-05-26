package legacy

import (
	"bytes"
	"math/big"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	newblock "github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	newtx "github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/dusk-network/dusk-wallet/v2/block"
	"github.com/dusk-network/dusk-wallet/v2/transactions"
	"github.com/sirupsen/logrus"
)

func oldBlockToNewBlock(b *block.Block) (*newblock.Block, error) {
	nb := newblock.NewBlock()
	nb.Header = oldHeaderToNewHeader(b.Header)
	calls, err := blockToContractCalls(b.Txs)
	if err != nil {
		return nil, err
	}

	nb.Txs = calls
	return nb, nil
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

func oldCertificateToNewCertificate(c *block.Certificate) *newblock.Certificate {
	nc := newblock.EmptyCertificate()
	nc.StepOneBatchedSig = c.StepOneBatchedSig
	nc.StepTwoBatchedSig = c.StepTwoBatchedSig
	nc.Step = c.Step
	nc.StepOneCommittee = c.StepOneCommittee
	nc.StepTwoCommittee = c.StepTwoCommittee
	return nc
}

func ProvisionersToRuskCommittee(p *user.Provisioners) []*rusk.Provisioner {
	ruskProvisioners := make([]*rusk.Provisioner, len(p.Members))
	i := 0
	for _, n := range p.Members {
		ruskProvisioners[i].BlsKey = n.PublicKeyBLS
		ruskProvisioners[i].Stakes = make([]*rusk.Stake, len(n.Stakes))
		for j, s := range n.Stakes {
			ruskProvisioners[i].Stakes[j].Amount = s.Amount
			ruskProvisioners[i].Stakes[j].EndHeight = s.EndHeight
			ruskProvisioners[i].Stakes[j].StartHeight = s.StartHeight
		}
		i++
	}

	return ruskProvisioners
}

func blockToContractCalls(txs []transactions.Transaction) ([]newtx.ContractCall, error) {
	calls := make([]newtx.ContractCall, len(txs))

	for i, c := range txs {
		switch c.Type() {
		case transactions.StandardType:
			call, err := StandardToRuskTx(c.(*transactions.Standard))
			if err != nil {
				return nil, err
			}

			tx := &newtx.Transaction{}
			if err := newtx.UTx(call, tx); err != nil {
				return nil, err
			}

			calls[i] = tx
		case transactions.StakeType:
			call, err := StakeToRuskStake(c.(*transactions.Stake))
			if err != nil {
				return nil, err
			}

			tx := &newtx.StakeTransaction{}
			if err := newtx.UStake(call, tx); err != nil {
				return nil, err
			}

			calls[i] = tx
		case transactions.BidType:
			call, err := BidToRuskBid(c.(*transactions.Bid))
			if err != nil {
				return nil, err
			}

			tx := &newtx.BidTransaction{}
			if err := newtx.UBid(call, tx); err != nil {
				return nil, err
			}

			calls[i] = tx
		case transactions.CoinbaseType:
			call, err := CoinbaseToRuskDistribute(c.(*transactions.Coinbase))
			if err != nil {
				return nil, err
			}

			tx := &newtx.DistributeTransaction{}
			if err := newtx.UDistribute(call, tx); err != nil {
				return nil, err
			}

			calls[i] = tx
		default:
			logrus.Warnln("encountered unexpected tx type")
			continue
		}
	}

	return calls, nil
}

func ContractCallsToBlock(calls []*rusk.ContractCallTx) (*block.Block, error) {
	blk := block.NewBlock()

	for _, call := range calls {
		var tx transactions.Transaction
		var err error
		switch call.ContractCall.(type) {
		case *rusk.ContractCallTx_Tx:
			tx, err = RuskTxToStandard(call.GetTx())
		case *rusk.ContractCallTx_Stake:
			tx, err = RuskStakeToStake(call.GetStake())
		case *rusk.ContractCallTx_Bid:
			tx, err = RuskBidToBid(call.GetBid())
		case *rusk.ContractCallTx_Distribute:
			tx, err = RuskDistributeToCoinbase(call.GetDistribute())
		default:
			logrus.Warnln("encountered unexpected tx type")
			continue
		}

		if err != nil {
			return nil, err
		}

		blk.AddTx(tx)
	}

	return blk, nil
}

func StandardToRuskTx(tx *transactions.Standard) (*rusk.Transaction, error) {
	buf := new(bytes.Buffer)
	if err := tx.RangeProof.Encode(buf, true); err != nil {
		return nil, err
	}

	inputs, err := inputsToRuskInputs(tx.Inputs)
	if err != nil {
		return nil, err
	}
	outputs := outputsToRuskOutputs(tx.Outputs)

	return &rusk.Transaction{
		Inputs:  inputs,
		Outputs: outputs,
		Fee: &rusk.TransactionOutput{
			BlindingFactor: &rusk.Scalar{
				Data: make([]byte, 32),
			},
			Pk: &rusk.PublicKey{
				AG: &rusk.CompressedPoint{
					Y: make([]byte, 32),
				},
				BG: &rusk.CompressedPoint{
					Y: make([]byte, 32),
				},
			},
			Value: tx.Fee.BigInt().Uint64(),
			Note: &rusk.Note{
				ValueCommitment: &rusk.Scalar{
					Data: make([]byte, 0),
				},
				RG: &rusk.CompressedPoint{
					Y: make([]byte, 0),
				},
				Nonce: &rusk.Nonce{
					Bs: make([]byte, 0),
				},
				PkR: &rusk.CompressedPoint{
					Y: make([]byte, 0),
				},
				BlindingFactor: &rusk.Note_TransparentBlindingFactor{
					TransparentBlindingFactor: &rusk.Scalar{Data: []byte{0x55, 0x66}},
				},
				Value: &rusk.Note_TransparentValue{
					TransparentValue: uint64(0),
				},
			},
		},
		Proof: buf.Bytes(),
		Data:  tx.R.Bytes(),
	}, nil
}

func RuskTxToStandard(tx *rusk.Transaction) (*transactions.Standard, error) {
	var feeScalar ristretto.Scalar
	feeScalar.SetBigInt(big.NewInt(int64(tx.Fee.Value)))

	var rPoint ristretto.Point
	var dataArr [32]byte
	copy(dataArr[:], tx.Data[0:32])
	rPoint.SetBytes(&dataArr)

	inputs, err := ruskInputsToInputs(tx.Inputs)
	if err != nil {
		return nil, err
	}
	outputs := ruskOutputsToOutputs(tx.Outputs)

	stx := &transactions.Standard{
		TxType:  transactions.StandardType,
		R:       rPoint,
		Inputs:  inputs,
		Outputs: outputs,
		Version: 0,
		Fee:     feeScalar,
	}

	if err := stx.RangeProof.Decode(bytes.NewBuffer(tx.Proof), true); err != nil {
		return nil, err
	}

	return stx, nil
}

func StakeToRuskStake(tx *transactions.Stake) (*rusk.StakeTransaction, error) {
	rtx, err := StandardToRuskTx(tx.StandardTx())
	if err != nil {
		return nil, err
	}

	return &rusk.StakeTransaction{
		BlsKey:           tx.PubKeyBLS,
		ExpirationHeight: tx.LockTime(),
		Tx:               rtx,
	}, nil
}

func RuskStakeToStake(tx *rusk.StakeTransaction) (*transactions.Stake, error) {
	stx, err := RuskTxToStandard(tx.Tx)
	if err != nil {
		return nil, err
	}

	return &transactions.Stake{
		Timelock: &transactions.Timelock{
			Standard: stx,
			Lock:     tx.ExpirationHeight,
		},
		PubKeyBLS: tx.BlsKey,
	}, nil
}

func BidToRuskBid(tx *transactions.Bid) (*rusk.BidTransaction, error) {
	rtx, err := StandardToRuskTx(tx.StandardTx())
	if err != nil {
		return nil, err
	}

	return &rusk.BidTransaction{
		Tx:               rtx,
		M:                tx.M,
		ExpirationHeight: tx.LockTime(),
	}, nil
}

func RuskBidToBid(tx *rusk.BidTransaction) (*transactions.Bid, error) {
	stx, err := RuskTxToStandard(tx.Tx)
	if err != nil {
		return nil, err
	}

	return &transactions.Bid{
		Timelock: &transactions.Timelock{
			Standard: stx,
			Lock:     tx.ExpirationHeight,
		},
		M: tx.M,
	}, nil
}

// TODO: implement
func RuskDistributeToCoinbase(tx *rusk.DistributeTransaction) (*transactions.Coinbase, error) {
	return nil, nil
}

// TODO: implement
func CoinbaseToRuskDistribute(cb *transactions.Coinbase) (*rusk.DistributeTransaction, error) {
	return nil, nil
}

func inputsToRuskInputs(inputs transactions.Inputs) ([]*rusk.TransactionInput, error) {
	rInputs := make([]*rusk.TransactionInput, len(inputs))

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

		sigBuf := new(bytes.Buffer)
		if err := input.Signature.Encode(sigBuf, true); err != nil {
			return nil, err
		}

		rInputs[i] = &rusk.TransactionInput{
			Nullifier: &rusk.Nullifier{
				H: &rusk.Scalar{
					Data: buf.Bytes(),
				},
			},
			MerkleRoot: &rusk.Scalar{
				Data: sigBuf.Bytes(),
			},
		}
	}

	return rInputs, nil
}

func ruskInputsToInputs(inputs []*rusk.TransactionInput) (transactions.Inputs, error) {
	sInputs := make(transactions.Inputs, len(inputs))

	for i, input := range inputs {
		buf := bytes.NewBuffer(input.Nullifier.H.Data)
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

		sigBuf := bytes.NewBuffer(input.MerkleRoot.Data)
		if err := sInputs[i].Signature.Decode(sigBuf, true); err != nil {
			return nil, err
		}
	}

	return sInputs, nil
}

func outputsToRuskOutputs(outputs transactions.Outputs) []*rusk.TransactionOutput {
	rOutputs := make([]*rusk.TransactionOutput, len(outputs))

	for i, output := range outputs {
		rOutputs[i] = &rusk.TransactionOutput{
			BlindingFactor: &rusk.Scalar{
				Data: output.Commitment.Bytes(),
			},
			Pk: &rusk.PublicKey{
				AG: &rusk.CompressedPoint{
					Y: output.PubKey.P.Bytes(),
				},
				BG: &rusk.CompressedPoint{
					Y: make([]byte, 32),
				},
			},
			Note: &rusk.Note{
				ValueCommitment: &rusk.Scalar{
					Data: output.EncryptedAmount.Bytes(),
				},
				RG: &rusk.CompressedPoint{
					Y: output.EncryptedMask.Bytes(),
				},
				Nonce: &rusk.Nonce{
					Bs: make([]byte, 0),
				},
				PkR: &rusk.CompressedPoint{
					Y: make([]byte, 0),
				},
				BlindingFactor: &rusk.Note_TransparentBlindingFactor{
					TransparentBlindingFactor: &rusk.Scalar{Data: []byte{0x55, 0x66}},
				},
				Value: &rusk.Note_TransparentValue{
					TransparentValue: uint64(0),
				},
			},
		}
	}

	return rOutputs
}

func ruskOutputsToOutputs(outputs []*rusk.TransactionOutput) transactions.Outputs {
	sOutputs := make(transactions.Outputs, len(outputs))

	for i, output := range outputs {
		_ = sOutputs[i].Commitment.UnmarshalBinary(output.BlindingFactor.Data)
		_ = sOutputs[i].PubKey.P.UnmarshalBinary(output.Pk.AG.Y)
		_ = sOutputs[i].EncryptedAmount.UnmarshalBinary(output.Note.ValueCommitment.Data)
		_ = sOutputs[i].EncryptedMask.UnmarshalBinary(output.Note.RG.Y)
	}
	return sOutputs
}

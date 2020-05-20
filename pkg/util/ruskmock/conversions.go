package ruskmock

import (
	"bytes"
	"math/big"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/dusk-network/dusk-wallet/v2/block"
	"github.com/dusk-network/dusk-wallet/v2/transactions"
)

func contractCallsToBlock([]*rusk.ContractCallTx) (*block.Block, error) {
	return nil, nil
}

func standardToRuskTx(tx *transactions.Standard) (*rusk.Transaction, error) {
	buf := new(bytes.Buffer)
	if err := tx.RangeProof.Encode(buf, true); err != nil {
		return nil, err
	}

	rtx := &rusk.Transaction{
		Inputs:  inputsToRuskInputs(tx.Inputs),
		Outputs: outputsToRuskOutputs(tx.Outputs),
		Fee: &rusk.TransactionOutput{
			Value: tx.Fee.BigInt().Uint64(),
		},
		Proof: buf.Bytes(),
		Data:  tx.R.Bytes(),
	}

	return rtx, nil
}

func ruskTxToStandard(tx *rusk.Transaction) (*transactions.Standard, error) {
	var feeScalar ristretto.Scalar
	feeScalar.SetBigInt(big.NewInt(int64(tx.Fee.Value)))

	var rPoint ristretto.Point
	var dataArr [32]byte
	copy(dataArr[:], tx.Data[0:32])
	rPoint.SetBytes(&dataArr)

	stx := &transactions.Standard{
		TxType:  transactions.StandardType,
		R:       rPoint,
		Inputs:  ruskInputsToInputs(tx.Inputs),
		Outputs: ruskOutputsToOutputs(tx.Outputs),
		Version: 0,
		Fee:     feeScalar,
	}

	if err := stx.RangeProof.Decode(bytes.NewBuffer(tx.Proof), true); err != nil {
		return nil, err
	}

	return stx, nil
}

func stakeToRuskStake(tx *transactions.Stake) (*rusk.StakeTransaction, error) {
	rtx, err := standardToRuskTx(tx.StandardTx())
	if err != nil {
		return nil, err
	}

	rStake := &rusk.StakeTransaction{
		BlsKey:           tx.PubKeyBLS,
		ExpirationHeight: tx.LockTime(),
		Tx:               rtx,
	}
	return rStake, nil

}

func ruskStakeToStake(tx *rusk.StakeTransaction) (*transactions.Stake, error) {
	stx, err := ruskTxToStandard(tx.Tx)
	if err != nil {
		return nil, err
	}

	stake := &transactions.Stake{
		Timelock: &transactions.Timelock{
			Standard: stx,
			Lock:     tx.ExpirationHeight,
		},
		PubKeyBLS: tx.BlsKey,
	}

	return stake, nil
}

func bidToRuskBid(tx *transactions.Bid) (*rusk.BidTransaction, error) {
	rtx, err := standardToRuskTx(tx.StandardTx())
	if err != nil {
		return nil, err
	}

	rBid := &rusk.BidTransaction{
		Tx:               rtx,
		M:                tx.M,
		ExpirationHeight: tx.LockTime(),
	}

	return rBid, nil

}

func ruskBidToBid(tx *rusk.BidTransaction) (*transactions.Bid, error) {
	stx, err := ruskTxToStandard(tx.Tx)
	if err != nil {
		return nil, err
	}

	bid := &transactions.Bid{
		Timelock: &transactions.Timelock{
			Standard: stx,
			Lock:     tx.ExpirationHeight,
		},
		M: tx.M,
	}

	return bid, nil

}

func inputsToRuskInputs(inputs transactions.Inputs) []*rusk.TransactionInput {

	return nil
}

func ruskInputsToInputs(inputs []*rusk.TransactionInput) transactions.Inputs {
	return nil

}

func outputsToRuskOutputs(outputs transactions.Outputs) []*rusk.TransactionOutput {
	return nil

}

func ruskOutputsToOutputs(outputs []*rusk.TransactionOutput) transactions.Outputs {
	return nil

}

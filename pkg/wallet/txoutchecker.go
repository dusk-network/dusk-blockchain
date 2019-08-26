package wallet

import (
	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
	wiretx "github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
)

// TxOutChecker holds all of the necessary data
// in order to check if an oputput was sent to a specified user
type TxOutChecker struct {
	encryptedValues bool
	R               ristretto.Point
	Outputs         transactions.Outputs
}

func NewTxOutChecker(blk block.Block) []TxOutChecker {
	txcheckers := make([]TxOutChecker, 0, len(blk.Txs))

	for _, tx := range blk.Txs {
		txchecker := TxOutChecker{
			encryptedValues: shouldEncryptValues(tx),
		}

		var RBytes [32]byte
		txR := tx.StandardTx().R
		copy(RBytes[:], txR.Bytes()[:])
		var R ristretto.Point
		R.SetBytes(&RBytes)
		txchecker.R = R

		txchecker.Outputs = tx.StandardTx().Outputs
		txcheckers = append(txcheckers, txchecker)
	}
	return txcheckers
}

func shouldEncryptValues(tx wiretx.Transaction) bool {
	switch tx.Type() {
	case wiretx.StandardType:
		return true
	case wiretx.TimelockType:
		return true
	case wiretx.BidType:
		return false
	case wiretx.StakeType:
		return false
	case wiretx.CoinbaseType:
		return false
	default:
		return true
	}
}

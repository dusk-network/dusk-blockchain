package wallet

import (
	"github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	wiretx "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/wallet/transactions"
)

// TxOutChecker holds all of the necessary data
// in order to check if an oputput was sent to a specified user
type TxOutChecker struct {
	encryptedValues bool
	R               ristretto.Point
	Outputs         []*transactions.Output
}

func NewTxOutChecker(blk block.Block) []TxOutChecker {
	txcheckers := make([]TxOutChecker, 0, len(blk.Txs))

	for _, tx := range blk.Txs {
		txchecker := TxOutChecker{
			encryptedValues: shouldEncryptValues(tx),
		}

		var RBytes [32]byte
		copy(RBytes[:], tx.StandardTX().R[:])
		var R ristretto.Point
		R.SetBytes(&RBytes)

		txchecker.R = R

		// Convert dusk-node outputs to dusk-wallet outputs
		outs := make([]*transactions.Output, 0, tx.StandardTX().Outputs.Len())
		for _, output := range tx.StandardTX().Outputs {

			outs = append(outs, transactions.OutputFromWire(*output))

		}
		txchecker.Outputs = outs

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

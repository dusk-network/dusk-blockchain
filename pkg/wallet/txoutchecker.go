package wallet

import (
	"github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/wallet/transactions"
)

// TxOutChecker holds all of the necessary data
// in order to check if an oputput was sent to a specified user
type TxOutChecker struct {
	R       ristretto.Point
	Outputs []*transactions.Output
}

func NewTxOutChecker(blk block.Block) []TxOutChecker {
	txcheckers := make([]TxOutChecker, 0, len(blk.Txs))

	for _, tx := range blk.Txs {
		txchecker := TxOutChecker{}

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

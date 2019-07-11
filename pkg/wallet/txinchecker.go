package wallet

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
)

type keyImage []byte

//TxInChecker contains the necessary information to
// deduce whether a user has spent a tx. This is just the keyImage.
type TxInChecker struct {
	keyImages []keyImage
}

func NewTxInChecker(blk block.Block) []TxInChecker {
	txcheckers := make([]TxInChecker, 0, len(blk.Txs))

	for _, tx := range blk.Txs {
		keyImages := make([]keyImage, 0)
		for _, input := range tx.StandardTX().Inputs {
			keyImages = append(keyImages, input.KeyImage)
		}
		txcheckers = append(txcheckers, TxInChecker{keyImages})
	}
	return txcheckers
}

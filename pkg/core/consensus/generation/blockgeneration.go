package generation

// import (
// 	"time"

// 	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
// 	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
// )

// // GenerateBlock will create and return a candidate block, containing the passed
// // seed signature, and the passed transactions. The block will point back to the
// // passed header.
// func GenerateBlock(signedSeed []byte, txs []*transactions.Stealth,
// 	prevBlockHeader *block.Header) (*block.Block, error) {

// 	blk := block.NewBlock()
// 	setBlockHeaderFields(blk, prevBlockHeader, signedSeed)

// 	// Add transactions
// 	for _, tx := range txs {
// 		blk.AddTx(tx)
// 	}

// 	if err := blk.SetRoot(); err != nil {
// 		return nil, err
// 	}

// 	if err := blk.SetHash(); err != nil {
// 		return nil, err
// 	}

// 	return blk, nil
// }

// func setBlockHeaderFields(blk *block.Block, prevBlockHeader *block.Header,
// 	signedSeed []byte) {

// 	blk.SetPrevBlock(prevBlockHeader)
// 	blk.Header.Seed = signedSeed
// 	blk.Header.Height = prevBlockHeader.Height + 1
// 	blk.Header.Timestamp = time.Now().Unix()
// }

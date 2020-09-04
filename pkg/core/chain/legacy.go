package chain

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func setupBidValues() error {
	// We can just use anything for the D and K, since the blind bid is currently
	// mocked.
	// NOTE: this should be changed if we choose to use the actual blind bid.
	_, db := heavy.CreateDBConnection()
	return db.Update(func(t database.Transaction) error {
		k, _ := crypto.RandEntropy(32)
		d, _ := crypto.RandEntropy(32)
		return t.StoreBidValues(d, k, make([]byte, 32), 250000)
	})
}

// ReconstructCommittee will fill in the committee members that are present from genesis.
func reconstructCommittee(p *user.Provisioners, b *block.Block) error {
	// We should properly reconstruct the committee, though. This is because the keys
	// need to match up with what's found in the wallet.
	for _, tx := range b.Txs {
		t := tx.Type()
		switch t {
		case transactions.Stake:
			// amountBytes := tx.(*transactions.Transaction).TxPayload.Notes[0].Commitment.Data
			// amount := binary.LittleEndian.Uint64(amountBytes[0:8])

			// TODO: add
			// if err := p.Add(t.BlsKey, amount, 0, t.ExpirationHeight); err != nil {
			// 	return fmt.Errorf("unexpected error in adding provisioner following a stake transaction: %v", err)
			// }
		}
	}
	return nil
}

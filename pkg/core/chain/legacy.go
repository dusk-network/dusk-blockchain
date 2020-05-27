package chain

import (
	"encoding/binary"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
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

func (c *Chain) reconstructCommittee(b *block.Block) error {
	// We should properly reconstruct the committee, though. This is because the keys
	// need to match up with what's found in the wallet.
	for _, tx := range b.Txs {
		switch t := tx.(type) {
		case *transactions.StakeTransaction:
			amountBytes := t.ContractTx.Tx.Outputs[0].Note.ValueCommitment.Data
			amount := binary.LittleEndian.Uint64(amountBytes[0:8])

			if err := c.addProvisioner(t.BlsKey, amount, 0, t.ExpirationHeight); err != nil {
				return fmt.Errorf("unexpected error in adding provisioner following a stake transaction: %v", err)
			}
		}
	}
	return nil
}

// addProvisioner will add a Member to the Provisioners by using the bytes of a BLS public key.
func (c *Chain) addProvisioner(pubKeyBLS []byte, amount, startHeight, endHeight uint64) error {
	if len(pubKeyBLS) != 129 {
		return fmt.Errorf("public key is %v bytes long instead of 129", len(pubKeyBLS))
	}

	i := string(pubKeyBLS)
	stake := user.Stake{Amount: amount, StartHeight: startHeight, EndHeight: endHeight}

	// Check for duplicates
	_, inserted := c.p.Set.IndexOf(pubKeyBLS)
	if inserted {
		// If they already exist, just add their new stake
		c.p.Members[i].AddStake(stake)
		return nil
	}

	// This is a new provisioner, so let's initialize the Member struct and add them to the list
	c.p.Set.Insert(pubKeyBLS)
	m := &user.Member{}

	m.PublicKeyBLS = pubKeyBLS
	m.AddStake(stake)

	c.p.Members[i] = m
	return nil
}

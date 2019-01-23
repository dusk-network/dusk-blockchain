package core

import (
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

// Provisioners is a mapping that links a public key to a node's staking info.
// A public key can be a BLS key or an Ed25519 key, they will end up pointing
// to the same Provisioner struct.
type Provisioners map[string]*Provisioner

// Provisioner defines a node's staking info.
type Provisioner struct {
	Stakes      []*transactions.Stealth
	BLSKey      *bls.PublicKey
	TotalAmount uint64
}

// SetupProvisioners will clear out all related entries and scan the blockchain
// for all staking transactions, logging the total weight and the information
// for each node.
func (b *Blockchain) SetupProvisioners() error {
	b.totalStakeWeight = b.stakeWeight             // Reset total weight
	b.provisioners = make(map[string]*Provisioner) // Reset provisioners

	l := maxLockTime
	if b.height < uint64(maxLockTime) {
		l = int(b.height)
	}

	currHeight := b.height
	for i := 0; i < l; i++ {
		hdr, err := b.db.GetBlockHeaderByHeight(currHeight)
		if err != nil {
			return err
		}

		block, err := b.db.GetBlock(hdr.Hash)
		if err != nil {
			return err
		}

		for _, v := range block.Txs {
			tx := v.(*transactions.Stealth)
			if tx.Type == transactions.StakeType {
				var amount uint64
				for _, output := range tx.Outputs {
					amount += output.Amount
				}

				b.AddProvisionerInfo(tx, amount)
				b.totalStakeWeight += amount
			}
		}

		currHeight--
	}

	return nil
}

// AddProvisionerInfo will add the tx info to a provisioner, and update their total
// stake amount.
func (b *Blockchain) AddProvisionerInfo(tx *transactions.Stealth, amount uint64) {
	// Set information on blockchain struct
	pk := hex.EncodeToString(tx.R) // TODO: implement a special stake type that includes an ed25519 public key
	// TODO: Add bls public key
	b.provisioners[pk].Stakes = append(b.provisioners[pk].Stakes, tx)
	b.provisioners[pk].TotalAmount += amount

	// Set information on context object
	b.ctx.NodeWeights[pk] = b.provisioners[pk].TotalAmount
	// TODO: Add bls public key
	// b.ctx.NodeKeys[pk] = tx.BLS
}

// UpdateProvisioners will run through all known nodes and check if they have
// expired stakes, and removing them if they do.
func (b *Blockchain) UpdateProvisioners() {
	// Loop through all nodes
	for pk, node := range b.provisioners {
		// Loop through all their stakes
		for i, tx := range node.Stakes {
			// Remove if expired
			if b.height > tx.LockTime {
				// Get tx amount and deduct it from the total amount
				var amount uint64
				for _, output := range tx.Outputs {
					amount += output.Amount
				}

				node.TotalAmount -= amount

				// Update context info as well
				b.ctx.NodeWeights[pk] = node.TotalAmount
				if node.TotalAmount == 0 {
					// Remove BLS public key
				}

				// Finally, cut the tx out of the array
				node.Stakes = append(node.Stakes[:i], node.Stakes[i+1:]...)
			}
		}
	}
}

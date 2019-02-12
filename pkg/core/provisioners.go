package core

import (
	"bytes"
	"encoding/hex"
	"sort"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

// Provisioners is a mapping that links an ed25519 public key to a node's
// staking info.
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
				pl := tx.TypeInfo.(*transactions.Stake)
				b.AddProvisionerInfo(tx, pl.Output.Amount)
				b.totalStakeWeight += pl.Output.Amount
			}
		}

		currHeight--
	}

	b.UpdateProvisioners()
	return nil
}

// AddProvisionerInfo will add the tx info to a provisioner, and update their total
// stake amount.
func (b *Blockchain) AddProvisionerInfo(tx *transactions.Stealth, amount uint64) {
	// Set information on blockchain struct
	info := tx.TypeInfo.(*transactions.Stake)
	pkEd := hex.EncodeToString(info.PubKeyEd)
	b.provisioners[pkEd].Stakes = append(b.provisioners[pkEd].Stakes, tx)
	b.provisioners[pkEd].TotalAmount += amount

	// Set information on context object
	b.ctx.NodeWeights[pkEd] = b.provisioners[pkEd].TotalAmount
	pkBLS := hex.EncodeToString(info.PubKeyBLS)
	b.ctx.NodeBLS[pkBLS] = info.PubKeyEd
	b.ctx.Committee = append(b.ctx.Committee, info.PubKeyEd)

	// Sort committee list
	b.SortProvisioners()
}

// UpdateProvisioners will run through all known nodes and check if they have
// expired stakes, and removing them if they do. It will also keep the context
// committee sorted and up to date.
func (b *Blockchain) UpdateProvisioners() {
	// Loop through all nodes
	for pk, node := range b.provisioners {
		// Loop through all their stakes
		for i, tx := range node.Stakes {
			// Remove if expired
			pl := tx.TypeInfo.(*transactions.Stake)
			if b.height <= pl.Timelock {
				continue
			}

			// Get tx amount and deduct it from the total amount
			node.TotalAmount -= pl.Output.Amount

			// Update context info as well
			b.ctx.NodeWeights[pk] = node.TotalAmount

			// Finally, cut the tx out of the array
			node.Stakes = append(node.Stakes[:i], node.Stakes[i+1:]...)

			// If node has no stake left, remove it from committee
			if node.TotalAmount > 0 {
				continue
			}

			for i, pkBytes := range b.ctx.Committee {
				pkStr := hex.EncodeToString(pkBytes)
				if pk == pkStr {
					b.ctx.Committee = append(b.ctx.Committee[:i], b.ctx.Committee[i+1:]...)
				}
			}
		}
	}

	// Sort committee list
	b.SortProvisioners()
}

// SortProvisioners will sort the Committee slice on the context object
// lexicographically.
func (b *Blockchain) SortProvisioners() {
	sort.SliceStable(b.ctx.Committee, func(i, j int) bool {
		return bytes.Compare(b.ctx.Committee[i], b.ctx.Committee[j]) < bytes.Compare(b.ctx.Committee[j], b.ctx.Committee[i])
	})
}

package consensus

import (
	"encoding/hex"
	"math"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

const maxLockTime = math.MaxUint16

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
func (c *Consensus) SetupProvisioners() error {
	c.totalStakeWeight = c.stakeWeight             // Reset total weight
	c.provisioners = make(map[string]*Provisioner) // Reset provisioners

	height := c.GetLatestHeight()

	l := maxLockTime
	if height < uint64(maxLockTime) {
		l = int(height)
	}

	for i := 0; i < l; i++ {
		hdr, err := c.GetBlockHeaderByHeight(height)
		if err != nil {
			return err
		}

		block, err := c.GetBlock(hdr.Hash)
		if err != nil {
			return err
		}

		for _, v := range block.Txs {
			tx := v.(*transactions.Stealth)
			if tx.Type == transactions.StakeType {
				pl := tx.TypeInfo.(*transactions.Stake)
				c.AddProvisionerInfo(tx, pl.Output.Amount)
				c.totalStakeWeight += pl.Output.Amount
			}
		}

		height--
	}

	c.RemoveOldProvisioners()
	return nil
}

// AddProvisionerInfo will add the tx info to a provisioner, and update their total
// stake amount.
func (c *Consensus) AddProvisionerInfo(tx *transactions.Stealth, amount uint64) error {
	// Set information on blockchain struct
	info := tx.TypeInfo.(*transactions.Stake)
	pkEd := hex.EncodeToString(info.PubKeyEd)
	c.provisioners[pkEd].Stakes = append(c.provisioners[pkEd].Stakes, tx)
	c.provisioners[pkEd].TotalAmount += amount

	// Set information on context object
	c.ctx.NodeWeights[pkEd] = c.provisioners[pkEd].TotalAmount
	pkBLS := hex.EncodeToString(info.PubKeyBLS)
	c.ctx.NodeBLS[pkBLS] = info.PubKeyEd
	if err := c.ctx.Committee.AddMember(info.PubKeyEd); err != nil {
		return err
	}

	// Sort committee list
	c.ctx.Committee.Sort()
	return nil
}

// RemoveOldProvisioners will run through all known nodes and check if they have
// expired stakes, and removing them if they do. It will also keep the context
// committee sorted and up to date.
func (c *Consensus) RemoveOldProvisioners() {
	// Loop through all nodes
	for pk, node := range c.provisioners {
		// Loop through all their stakes
		for i, tx := range node.Stakes {
			// Remove if expired
			pl := tx.TypeInfo.(*transactions.Stake)
			if c.GetLatestHeight() <= pl.Timelock {
				continue
			}

			// Get tx amount and deduct it from the total amount
			node.TotalAmount -= pl.Output.Amount

			// Update context info as well
			c.ctx.NodeWeights[pk] = node.TotalAmount

			// Finally, cut the tx out of the array
			node.Stakes = append(node.Stakes[:i], node.Stakes[i+1:]...)

			// If node has no stake left, remove it from committee
			if node.TotalAmount > 0 {
				continue
			}

			for _, member := range *c.ctx.Committee {
				if pk == member.String() {
					c.ctx.Committee.RemoveMember(member[:])
				}
			}
		}
	}

	// Sort committee list
	c.ctx.Committee.Sort()
}

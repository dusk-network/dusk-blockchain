// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
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
		indexStoredBidBytes, _ := crypto.RandEntropy(8)
		return t.StoreBidValues(d, k, binary.LittleEndian.Uint64(indexStoredBidBytes), 250000)
	})
}

// ReconstructCommittee will fill in the committee members that are present from genesis.
//nolint
func reconstructCommittee(p *user.Provisioners, b *block.Block) error {
	// We should properly reconstruct the committee, though. This is because the keys
	// need to match up with what's found in the wallet.
	for _, tx := range b.Txs {
		t := tx.Type()
		switch t {
		case transactions.Stake:
			amountBytes := tx.(*transactions.Transaction).TxPayload.Notes[0].Commitment.Data
			amount := binary.LittleEndian.Uint64(amountBytes[0:8])
			buf := bytes.NewBuffer(tx.(*transactions.Transaction).TxPayload.CallData)
			var expirationHeight uint64
			if err := encoding.ReadUint64LE(buf, &expirationHeight); err != nil {
				return err
			}

			blsKey := make([]byte, 0)
			if err := encoding.ReadVarBytes(buf, &blsKey); err != nil {
				return err
			}

			if err := p.Add(blsKey, amount, 0, expirationHeight); err != nil {
				return fmt.Errorf("unexpected error in adding provisioner following a stake transaction: %v", err)
			}
		}
	}
	return nil
}

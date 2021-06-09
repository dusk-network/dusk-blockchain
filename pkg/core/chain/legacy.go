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
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

// ReconstructCommittee will fill in the committee members that are present from genesis.
//nolint
func ReconstructCommittee(p *user.Provisioners, b *block.Block) error {
	// We should properly reconstruct the committee, though. This is because the keys
	// need to match up with what's found in the wallet.
	for _, tx := range b.Txs {
		t := tx.Type()
		switch t {
		case transactions.Stake:
			amountBytes := tx.(*transactions.Transaction).Payload.Notes[0].Commitment
			amount := binary.LittleEndian.Uint64(amountBytes[0:8])
			buf := bytes.NewBuffer(tx.(*transactions.Transaction).Payload.CallData)

			var expirationHeight uint64
			if err := encoding.ReadUint64LE(buf, &expirationHeight); err != nil {
				return err
			}

			// Add grace period for stakes that aren't included in genesis.
			if b.Header.Height != 0 {
				expirationHeight += 1000
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

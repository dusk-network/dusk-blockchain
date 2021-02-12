// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package dupemap

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

var defaultTolerance uint64 = 3

// TODO: DupeMap should deal with value bytes.Buffer rather than pointers as it is not supposed to mutate the struct.
//nolint:golint
type DupeMap struct {
	round     uint64
	tmpMap    *TmpMap
	tolerance uint64
}

// Launch returns a dupemap which is self-cleaning, by launching a goroutine
// which listens for accepted blocks, and updates the height upon receipt.
func Launch(eventBus eventbus.Broker) *DupeMap {
	acceptedBlockChan, _ := consensus.InitAcceptedBlockUpdate(eventBus)
	dupeBlacklist := NewDupeMap(1, 300*1000)

	go func() {
		for {
			blk := <-acceptedBlockChan
			// NOTE: do we need locking?
			dupeBlacklist.UpdateHeight(blk.Header.Height)
		}
	}()

	return dupeBlacklist
}

// NewDupeMap returns a DupeMap.
func NewDupeMap(round uint64, capacity uint) *DupeMap {
	tmpMap := NewTmpMap(defaultTolerance, capacity)

	return &DupeMap{
		round,
		tmpMap,
		defaultTolerance,
	}
}

// UpdateHeight for a round.
func (d *DupeMap) UpdateHeight(round uint64) {
	d.tmpMap.UpdateHeight(round)
}

// SetTolerance for a round.
func (d *DupeMap) SetTolerance(roundNr uint64) {
	threshold := d.tmpMap.Height() - roundNr
	d.tmpMap.DeleteBefore(threshold)
	d.tmpMap.SetTolerance(roundNr)
}

// CanFwd tests if any of Cuckoo Filters (a filter per round) knows already
// this payload. Similarly to Bloom Filters, False positive matches are
// possible, but false negatives are not.
func (d *DupeMap) CanFwd(payload *bytes.Buffer) bool {
	found := d.tmpMap.HasAnywhere(payload)
	if found {
		return false
	}

	return d.tmpMap.Add(payload)
}

// Size returns the bytes count allocated by the underlying filter.
func (d *DupeMap) Size() int {
	return d.tmpMap.Size()
}

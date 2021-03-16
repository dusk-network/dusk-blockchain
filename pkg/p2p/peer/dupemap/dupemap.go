// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package dupemap

import (
	"bytes"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

const (
	defaultTolerance = uint64(3)
	defaultCapacity  = uint32(100000)
)

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

	capacity := cfg.Get().Network.MaxDupeMapItems
	if capacity == 0 {
		capacity = defaultCapacity
	}

	log.WithField("cap", capacity).Info("launch dupemap instance")

	// NB defaultTolerance is number of rounds before data expires. That's
	// said, the overall number of items per DupeMap instance is
	// defaultTolerance*maxItemsPerRound
	//
	// E.g 3 * 300,000 ~ 1MB max memory allocated by a DupeMap instance

	dupeBlacklist := NewDupeMap(1, capacity)

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
func NewDupeMap(round uint64, capacity uint32) *DupeMap {
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
	// Reset any bloom filters that have expired
	d.tmpMap.CleanExpired()

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

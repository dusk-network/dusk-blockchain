// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package dupemap

import (
	"bytes"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	log "github.com/sirupsen/logrus"
)

const (
	defaultTolerance = uint64(3)
	defaultCapacity  = uint32(100000)
	defaultExpire    = int64(5)
)

// TODO: DupeMap should deal with value bytes.Buffer rather than pointers as it is not supposed to mutate the struct.
//nolint:golint
type DupeMap struct {
	round     uint64
	tmpMap    *TmpMap
	tolerance uint64
}

// NewDupeMapDefault returns a dupemap instance with default config.
func NewDupeMapDefault() *DupeMap {
	capacity := cfg.Get().Network.MaxDupeMapItems
	if capacity == 0 {
		capacity = defaultCapacity
	}

	return NewDupeMap(defaultExpire, capacity)
}

// NewDupeMap creates new dupemap instance.
func NewDupeMap(expire int64, capacity uint32) *DupeMap {
	log.WithField("cap", capacity).Info("create dupemap instance")

	round := uint64(1)
	tmpMap := NewTmpMap(defaultTolerance, capacity, expire)

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

// HasAnywhere tests if any of Cuckoo Filters (a filter per round) knows already
// this payload. Similarly to Bloom Filters, False positive matches are
// possible, but false negatives are not.
func (d *DupeMap) HasAnywhere(payload *bytes.Buffer) bool {
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

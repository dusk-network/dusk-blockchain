// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package dupemap

import (
	"bytes"
	"sync"
	"time"

	cuckoo "github.com/seiflotfy/cuckoofilter"
)

type cache struct {
	*cuckoo.Filter
	TTL int64
}

type (
	//nolint:golint
	TmpMap struct {
		lock sync.RWMutex
		// current height
		height    uint64
		tolerance uint64

		// expire number of seconds for a cache before being reset
		expire int64

		// point in time current height will expire
		expiryTimestamp int64

		// map round to cuckoo filter
		msgFilter map[uint64]*cache
		capacity  uint32
	}
)

// NewTmpMap creates a TmpMap instance.
func NewTmpMap(tolerance uint64, capacity uint32, expire int64) *TmpMap {
	return &TmpMap{
		msgFilter: make(map[uint64]*cache),
		capacity:  capacity,
		height:    0,
		tolerance: tolerance,
		expire:    expire,
	}
}

// UpdateHeight for a round.
func (t *TmpMap) UpdateHeight(round uint64) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.height > round {
		return
	}

	_, found := t.msgFilter[round]
	if !found {
		t.msgFilter[round] = &cache{
			Filter: cuckoo.NewFilter(uint(t.capacity)),
			TTL:    time.Now().Unix() + t.expire,
		}
		t.height = round
		t.clean()
	}
}

//nolint:golint
func (t *TmpMap) Height() uint64 {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.height
}

//nolint:golint
func (t *TmpMap) Has(b *bytes.Buffer) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.has(b, t.height)
}

// HasAnywhere checks if the TmpMap contains a hash of the passed buffer at any height.
func (t *TmpMap) HasAnywhere(b *bytes.Buffer) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for k := range t.msgFilter {
		if t.has(b, k) {
			return true
		}
	}

	return false
}

// HasAt checks if the TmpMap contains a hash of the passed buffer at a specified height.
func (t *TmpMap) HasAt(b *bytes.Buffer, heigth uint64) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.has(b, heigth)
}

// DeleteBefore clears a Map of items stored before the given height.
func (t *TmpMap) DeleteBefore(height uint64) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.deleteBefore(height)
}

func (t *TmpMap) deleteBefore(height uint64) {
	currentMin := t.height - t.tolerance
	if currentMin >= height {
		return
	}

	for level := range t.msgFilter {
		if level < height {
			t.msgFilter[level].Reset()
			delete(t.msgFilter, level)
		}
	}
}

// SetTolerance adjusts how long hashes stay in the TmpMap until they are deleted.
func (t *TmpMap) SetTolerance(tolerance uint64) {
	t.lock.Lock()
	defer t.lock.Unlock()

	threshold := t.height - tolerance

	t.deleteBefore(threshold)
}

func (t *TmpMap) has(b *bytes.Buffer, heigth uint64) bool {
	f := t.msgFilter[heigth]
	if f == nil {
		return false
	}

	return f.Lookup(b.Bytes())
}

// Add the hash of a buffer to the blacklist.
// Returns true if the element was added. False otherwise.
func (t *TmpMap) Add(b *bytes.Buffer) bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.add(b, t.height)
}

// AddAt adds a hash of a buffer at a specific height.
func (t *TmpMap) AddAt(b *bytes.Buffer, height uint64) bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.add(b, height)
}

// Size returns overall size of all filters.
func (t *TmpMap) Size() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	var fullSize int
	for _, f := range t.msgFilter {
		fullSize += len(f.Encode())
	}

	return fullSize
}

// IsExpired returns true if TmpMap has expired.
func (t *TmpMap) IsExpired() bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return time.Now().Unix() >= t.expiryTimestamp
}

// clean the TmpMap up to the upto argument.
func (t *TmpMap) clean() {
	if t.height <= t.tolerance {
		// don't clean
		return
	}

	for r := range t.msgFilter {
		if r <= t.height-t.tolerance {
			t.msgFilter[r].Reset()
			delete(t.msgFilter, r)
		}
	}
}

// CleanExpired resets all cache instances that has expired.
func (t *TmpMap) CleanExpired() {
	t.lock.Lock()
	defer t.lock.Unlock()

	for height, f := range t.msgFilter {
		if time.Now().Unix() >= f.TTL {
			t.msgFilter[height].Reset()
			delete(t.msgFilter, height)
		}
	}
}

// add an entry to the set at the current height. Returns false if the element has not been added (due to being a duplicate).
func (t *TmpMap) add(b *bytes.Buffer, round uint64) bool {
	_, found := t.msgFilter[round]
	if !found {
		t.msgFilter[round] = &cache{
			Filter: cuckoo.NewFilter(uint(t.capacity)),
			TTL:    time.Now().Unix() + t.expire,
		}
	}

	return t.msgFilter[round].Insert(b.Bytes())
}

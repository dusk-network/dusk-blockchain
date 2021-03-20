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

		// expire number of seconds for a cache before being reset
		expire int64

		// point in time current height will expire
		expiryTimestamp int64

		// cuckoo filter
		msgFilter *cache
		capacity  uint32
	}
)

// NewTmpMap creates a TmpMap instance.
func NewTmpMap(capacity uint32, expire int64) *TmpMap {
	return &TmpMap{
		msgFilter: nil,
		capacity:  capacity,
		expire:    expire,
	}
}

//nolint:golint
func (t *TmpMap) Has(b *bytes.Buffer) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.has(b)
}

func (t *TmpMap) has(b *bytes.Buffer) bool {
	if t.msgFilter == nil {
		return false
	}

	return t.msgFilter.Lookup(b.Bytes())
}

// Add the hash of a buffer to the blacklist.
// Returns true if the element was added. False otherwise.
func (t *TmpMap) Add(b *bytes.Buffer) bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.add(b)
}

// Size returns overall size of all filters.
func (t *TmpMap) Size() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if t.msgFilter == nil {
		return 0
	}

	return len(t.msgFilter.Encode())
}

// IsExpired returns true if TmpMap has expired.
func (t *TmpMap) IsExpired() bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return time.Now().Unix() >= t.expiryTimestamp
}

// CleanExpired resets all cache instances that has expired.
func (t *TmpMap) CleanExpired() {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.msgFilter != nil {
		if time.Now().Unix() >= t.msgFilter.TTL {
			t.msgFilter.Reset()
			t.msgFilter = nil
		}
	}
}

// add an entry. Returns false if the element has not been added (due to being a duplicate).
func (t *TmpMap) add(b *bytes.Buffer) bool {
	if t.msgFilter == nil {
		t.msgFilter = &cache{
			Filter: cuckoo.NewFilter(uint(t.capacity)),
			TTL:    time.Now().Unix() + t.expire,
		}
	}

	return t.msgFilter.Insert(b.Bytes())
}

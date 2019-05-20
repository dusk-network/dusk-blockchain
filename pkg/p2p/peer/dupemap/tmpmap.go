package dupemap

import (
	"bytes"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/hashset"
)

type (
	TmpMap struct {
		lock      sync.RWMutex
		height    uint64
		msgSets   map[uint64]*hashset.Set
		tolerance uint64
	}
)

func NewTmpMap(tolerance uint64) *TmpMap {
	msgSets := make(map[uint64]*hashset.Set)
	return &TmpMap{
		msgSets:   msgSets,
		height:    0,
		tolerance: tolerance,
	}
}

func (t *TmpMap) UpdateHeight(round uint64) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.height > round {
		return
	}
	_, found := t.msgSets[round]
	if !found {
		t.msgSets[round] = hashset.New()
		t.height = round
		t.clean()
	}
}

func (t *TmpMap) Height() uint64 {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.height
}

func (t *TmpMap) Has(b *bytes.Buffer) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.has(b, t.height)
}

// HasAnywhere checks if the TmpMap contains a hash of the passed buffer at any height.
func (t *TmpMap) HasAnywhere(b *bytes.Buffer) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	for k := range t.msgSets {
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

// DeleteBefore clears a Map of
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

	for level := range t.msgSets {
		if level < height {
			delete(t.msgSets, level)
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
	set := t.msgSets[heigth]
	if set == nil {
		return false
	}
	return set.Has(b.Bytes())
}

// Add the hash of a buffer to the blacklist.
// Returns true if the element was added. False otherwise
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

// Clean the TmpMap up to the upto argument
func (t *TmpMap) clean() {
	if t.height <= t.tolerance {
		// don't clean
		return
	}

	for r := range t.msgSets {
		if r <= t.height-t.tolerance {
			delete(t.msgSets, r)
		}
	}
}

// add an entry to the set at the current height. Returns false if the element has not been added (due to being a duplicate)
func (t *TmpMap) add(b *bytes.Buffer, round uint64) bool {
	set, found := t.msgSets[round]
	if !found {
		set = hashset.New()
	}

	ret := set.Add(b.Bytes())
	t.msgSets[round] = set
	return ret
}

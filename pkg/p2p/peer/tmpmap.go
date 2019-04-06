package peer

import (
	"bytes"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

type (
	TmpMap struct {
		lock      sync.RWMutex
		msgSets   map[uint64]*hashSet
		height    uint64
		tolerance uint64
	}

	hashSet struct {
		set map[string]bool
	}
)

func NewTmpMap(tolerance uint64) *TmpMap {
	msgSets := make(map[uint64]*hashSet)

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
		t.msgSets[round] = newSet()
		t.height = round
		t.clean()
	}
}

func (t *TmpMap) Has(b *bytes.Buffer) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.has(b, t.height)
}

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

func (t *TmpMap) HasAt(b *bytes.Buffer, heigth uint64) bool {

	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.has(b, heigth)
}

func (t *TmpMap) has(b *bytes.Buffer, heigth uint64) bool {
	h, _ := hash.Xxhash(b.Bytes())
	hs := string(h)

	set := t.msgSets[heigth]
	if set == nil {
		return false
	}
	return set.has(hs)
}

// Add the hash of a buffer to the blacklist.
// Returns true if the element was added. False otherwise
func (t *TmpMap) Add(b *bytes.Buffer) bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.add(b, t.height)
}

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
		set = newSet()
	}

	ret := set.add(b.Bytes())
	t.msgSets[round] = set
	return ret
}

func newSet() *hashSet {
	return &hashSet{
		make(map[string]bool),
	}
}

func (s *hashSet) has(k string) bool {
	_, found := s.set[k]
	return found
}

// add the xxH64 hash of some data to the set. Returns false if the entry was already there
func (s *hashSet) add(data []byte) bool {
	hashed, err := hash.Xxhash(data)
	if err != nil {
		return false
	}
	hs := string(hashed)
	_, found := s.set[hs]
	s.set[hs] = true
	return found
}

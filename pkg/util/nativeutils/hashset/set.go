// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package hashset

import (
	"sync"

	"github.com/dusk-network/dusk-crypto/hash"
)

var _empty = new(struct{})

// SafeSet is a threadsafe hashset.
type SafeSet struct {
	sync.RWMutex
	*Set
}

// NewSafe returns an initialized SafeSet.
func NewSafe() *SafeSet {
	return &SafeSet{
		Set: New(),
	}
}

// Has returns true if the entry is found.
func (s *SafeSet) Has(data []byte) bool {
	s.RLock()
	defer s.RUnlock()
	return s.Set.Has(data)
}

// Add an entry to the current set. If the entry is already there, it returns true.
func (s *SafeSet) Add(data []byte) bool {
	s.Lock()
	defer s.Unlock()
	return s.Set.Add(data)
}

// Size returns the number of elements in the SafeSet.
func (s *SafeSet) Size() int {
	s.RLock()
	defer s.RUnlock()
	return s.Set.Size()
}

// Remove returns the number of elements in the SafeSet.
func (s *SafeSet) Remove(data []byte) bool {
	s.Lock()
	defer s.Unlock()
	return s.Set.Remove(data)
}

// Set is a hashset implementation. The hashing function is Xxhash for its high performance.
type Set struct {
	entries map[string]*struct{}
}

// New creates a new Set.
func New() *Set {
	return &Set{
		entries: make(map[string]*struct{}),
	}
}

// Has returns true if the entry is found.
func (s *Set) Has(data []byte) bool {
	hs := repr(data)
	return s != nil && s.has(hs)
}

// Add an entry to the current set. If the entry is already there, it returns true.
func (s *Set) Add(data []byte) bool {
	hs := repr(data)
	_, found := s.entries[hs]
	s.entries[hs] = _empty
	return found
}

// Remove an entry from the set. Returns true if the entry was found. False
// otherwise.
func (s *Set) Remove(data []byte) bool {
	hs := repr(data)

	_, found := s.entries[hs]
	if found {
		delete(s.entries, hs)
		return true
	}

	return false
}

func (s *Set) has(k string) bool {
	_, found := s.entries[k]
	return found
}

// Size returns the number of elements in the Set.
func (s *Set) Size() int {
	return len(s.entries)
}

func repr(data []byte) string {
	hashed, _ := hash.Xxhash(data)
	return string(hashed)
}

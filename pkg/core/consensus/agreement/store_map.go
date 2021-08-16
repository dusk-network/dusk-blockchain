// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package agreement

import "sync"

type hash [32]byte

type storeMap struct {
	sync.RWMutex
	stores map[hash]*store
}

func newStoreMap() *storeMap {
	return &storeMap{stores: make(map[hash]*store)}
}

func (m *storeMap) getStoreByHash(blkHash []byte) *store {
	m.RLock()
	defer m.RUnlock()

	var (
		h  hash
		ok bool
		r  *store
	)

	copy(h[:], blkHash)

	if r, ok = m.stores[h]; !ok {
		return nil
	}

	return r
}

func (m *storeMap) makeStoreByHash(blkHash []byte) *store {
	m.Lock()
	defer m.Unlock()

	var h hash
	copy(h[:], blkHash)

	st := newStore()
	m.stores[h] = st

	return st
}

func (m *storeMap) len() int {
	m.RLock()
	defer m.RUnlock()

	return len(m.stores)
}

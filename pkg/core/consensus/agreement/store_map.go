// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package agreement

type hash [32]byte

type storeMap map[hash]*store

func newStoreMap() storeMap {
	return make(map[hash]*store)
}

func (s storeMap) getStoreByHash(blkHash []byte) *store {
	var (
		h  hash
		ok bool
		r  *store
	)

	copy(h[:], blkHash)

	if r, ok = s[h]; !ok {
		return nil
	}

	return r
}

func (s storeMap) makeStoreByHash(blkHash []byte) *store {
	var h hash
	copy(h[:], blkHash)

	st := newStore()
	s[h] = st

	return st
}

func (s storeMap) len() int {
	return len(s)
}

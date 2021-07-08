// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package sortedset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/dusk-network/dusk-blockchain/pkg/util"
)

// All represents the bitmap of the whole set.
const All uint64 = math.MaxUint64

// Set is an ordered set of big.Int.
type Set []*big.Int

// Len complies with the Sort interface.
func (v Set) Len() int { return len(v) }

// Swap complies with the Sort interface.
func (v Set) Swap(i, j int) { v[i], v[j] = v[j], v[i] }

// Less complies with the Sort interface.
func (v Set) Less(i, j int) bool { return v[i].Cmp(v[j]) < 0 }

// New creates a new instance of a sorted set.
func New() Set {
	return make([]*big.Int, 0)
}

// Equal tests a set for equality.
func (v Set) Equal(other Set) bool {
	sort.Sort(other)

	for i := 0; i < len(v); i++ {
		if v[i].Cmp(other[i]) != 0 {
			return false
		}
	}

	return true
}

// Copy deeply a set.
func (v Set) Copy() Set {
	all := make([]*big.Int, len(v))
	for i, elem := range v {
		all[i] = new(big.Int)
		all[i].Set(elem)
	}

	return Set(all)
}

// IndexOf returns the index at which a byte slice should be inserted and
// whether the element is actually found or otherwise. Internally uses big.Int
// representation.
func (v Set) IndexOf(b []byte) (int, bool) {
	// trivially handle the case of an empty set
	if len(v) == 0 {
		return 0, false
	}

	// turn the []byte into a big.Int
	iPk := new(big.Int).SetBytes(b)
	return v.indexOf(iPk)
}

// indexOf returns the index at which a big.Int representation of a BLS key
// should be inserted and whether the key is actually present or otherwise.
func (v Set) indexOf(iPk *big.Int) (int, bool) {
	// use binary search to get the index of the element
	idx := sort.Search(len(v), func(i int) bool {
		return v[i].Cmp(iPk) >= 0
	})

	if idx < len(v) && iPk.Cmp(v[idx]) == 0 {
		// the element is actually found at the index
		return idx, true
	}

	// the element wasn't found
	return idx, false
}

// Insert a big.Int representation of a BLS key at a proper index (respectful of the VotingCommittee order).
// If the element is already in the VotingCommittee does nothing and returns false.
func (v *Set) Insert(b []byte) bool {
	iRepr := new(big.Int).SetBytes(b)

	l := len(*v)
	if l == 0 {
		*v = append(*v, iRepr)
		return true
	}

	idx, found := v.indexOf(iRepr)
	if found {
		return false
	}

	*v = append(*v, new(big.Int))
	copy((*v)[idx+1:], (*v)[idx:])

	(*v)[idx] = iRepr
	return true
}

// Remove an entry from the set. Return false if the entry can't be found.
func (v *Set) Remove(pubKeyBLS []byte) bool {
	i, found := v.IndexOf(pubKeyBLS)
	if found {
		// TODO: this is inefficient
		*v = append((*v)[:i], (*v)[i+1:]...)
		return true
	}

	return false
}

// Intersect the bit representation of a VotingCommittee subset with the whole VotingCommittee set.
func (v Set) Intersect(committeeSet uint64) Set {
	if committeeSet == All || committeeSet == v.Whole() {
		return v[:]
	}

	c := New()

	for i, elem := range v {
		// looping on all bits to see which one is set to 1
		if ((committeeSet >> uint(i)) & 1) != 0 {
			c = append(c, elem)
		}
	}

	return c
}

// Bits creates a bit representation of the subset of a Set. The subset is passed by value.
func (v *Set) Bits(subset Set) uint64 {
	ret := uint64(0)

	if len(subset) == 0 {
		return ret
	}

	for _, cmpElem := range subset {
		for i, elem := range *v {
			if bytes.Equal(elem.Bytes(), cmpElem.Bytes()) {
				ret |= 1 << uint(i) // flip the i-th bit to 1
				break
			}
		}
	}

	return ret
}

func (v Set) String() string {
	var str strings.Builder

	for i, bi := range v {
		_, _ = str.WriteString("idx: ")
		_, _ = str.WriteString(strconv.Itoa(i))
		_, _ = str.WriteString(" nr: ")
		_, _ = str.WriteString(shortStr(bi))
		_, _ = str.WriteString("\n")
	}

	return str.String()
}

func shortStr(i *big.Int) string {
	var str strings.Builder

	iStr := i.String()
	_, _ = str.WriteString(iStr[:3])
	_, _ = str.WriteString("...")
	_, _ = str.WriteString(iStr[len(iStr)-3:])

	return str.String()
}

// Format implements fmt.Formatter interface.
func (v Set) Format(f fmt.State, c rune) {
	for _, bi := range v {
		r := fmt.Sprintf("Key: %s", util.StringifyBytes(bi.Bytes()))
		_, _ = f.Write([]byte(r))
	}
}

// MarshalJSON ...
func (v Set) MarshalJSON() ([]byte, error) {
	data := make([]string, 0)

	for _, bi := range v {
		r := fmt.Sprintf("Key: %s", util.StringifyBytes(bi.Bytes()))
		data = append(data, r)
	}

	return json.Marshal(data)
}

// Whole returns the bitmap of all the elements within the set.
func (v Set) Whole() uint64 {
	ret := uint64(0)
	for i := range v {
		ret |= 1 << uint(i)
	}

	return ret
}

// Contains returns true if Set contains b.
func (v *Set) Contains(b []byte) bool {
	iRepr := new(big.Int).SetBytes(b)
	_, found := v.indexOf(iRepr)
	return found
}

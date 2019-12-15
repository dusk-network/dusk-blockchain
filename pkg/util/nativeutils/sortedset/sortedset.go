package sortedset

import (
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

const All uint64 = math.MaxUint64

type Set []*big.Int

func (v Set) Len() int           { return len(v) }
func (v Set) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v Set) Less(i, j int) bool { return v[i].Cmp(v[j]) < 0 }

func New() Set {
	return make([]*big.Int, 0)
}

func (v Set) Equal(other Set) bool {
	sort.Sort(other)
	for i := 0; i < len(v); i++ {
		if v[i].Cmp(other[i]) != 0 {
			return false
		}
	}

	return true
}

// IndexOf returns the index at which a byte slice should be inserted and
// whether the element is actually found or otherwise. Internally uses big.Int
// representation
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
// should be inserted and whether the
// key is actually present or otherwise
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
// If the element is already in the VotingCommittee does nothing and returns false
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

// Remove an entry from the set. Return false if the entry can't be found
func (v *Set) Remove(pubKeyBLS []byte) bool {
	i, found := v.IndexOf(pubKeyBLS)
	if found {
		// TODO: this is inefficient
		*v = append((*v)[:i], (*v)[i+1:]...)
		return true
	}
	return false
}

// Intersect the bit representation of a VotingCommittee subset with the whole VotingCommittee set
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

// Bits creates a bit representation of the subset of a Set. The subset is passed by value
func (v *Set) Bits(subset Set) uint64 {
	ret := uint64(0)
	if len(subset) == 0 {
		return ret
	}

	var head *big.Int
	head, subset = subset[0], subset[1:]
	for i, elem := range *v {
		if elem.Cmp(head) == 0 {
			ret |= (1 << uint(i)) // flip the i-th bit to 1
			if len(subset) == 0 {
				break
			}
			head, subset = subset[0], subset[1:]
		}
	}
	return ret
}

func (v Set) String() string {
	var str strings.Builder
	for i, bi := range v {
		str.WriteString("idx: ")
		str.WriteString(strconv.Itoa(i))
		str.WriteString(" nr: ")
		str.WriteString(shortStr(bi))
		str.WriteString("\n")
	}
	return str.String()
}

func (v Set) Whole() uint64 {
	ret := uint64(0)
	for i := range v {
		ret |= (1 << uint(i))
	}
	return ret
}

func shortStr(i *big.Int) string {
	var str strings.Builder
	iStr := i.String()
	str.WriteString(iStr[:3])
	str.WriteString("...")
	str.WriteString(iStr[len(iStr)-3:])
	return str.String()
}

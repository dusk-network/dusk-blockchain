package sortedset

import (
	"math/big"
	"sort"
	"strconv"
	"strings"
)

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

func (v Set) indexOf(k *big.Int) (int, bool) {
	cmp := 0
	found := false

	i := sort.Search(len(v), func(i int) bool {
		cmp = v[i].Cmp(k)
		if cmp == 0 {
			found = true
		}
		return cmp >= 0
	})
	return i, found
}

func (v Set) IndexOf(b []byte) (int, bool) {
	iPk := &big.Int{}
	iPk.SetBytes(b)
	return v.indexOf(iPk)
}

// Remove an entry from the set. Return false if the entry can't be found
func (v *Set) Remove(pubKeyBLS []byte) bool {
	i, found := v.IndexOf(pubKeyBLS)
	if found {
		*v = append((*v)[:i], (*v)[i+1:]...)
		return true
	}
	return false
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

// Insert a big.Int representation of a BLS key at a proper index (respectful of the VotingCommittee order). If the element is already in the VotingCommittee does nothing and returns false
func (v *Set) Insert(b []byte) bool {
	i, found := v.IndexOf(b)
	if !found {
		iRepr := &big.Int{}
		iRepr.SetBytes(b)
		*v = append((*v)[:i], append([]*big.Int{iRepr}, (*v)[i:]...)...)
		return true
	}
	return false
}

func shortStr(i *big.Int) string {
	var str strings.Builder
	iStr := i.String()
	str.WriteString(iStr[:3])
	str.WriteString("...")
	str.WriteString(iStr[len(iStr)-3:])
	return str.String()
}

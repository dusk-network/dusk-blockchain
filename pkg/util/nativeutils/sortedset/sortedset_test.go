package sortedset

import (
	"math/big"
	"math/rand"
	"sort"
	"testing"

	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestBits(t *testing.T) {
	set := New()
	subset := New()

	set = append(set, big.NewInt(0))
	set = append(set, big.NewInt(10))
	set = append(set, big.NewInt(20))
	set = append(set, big.NewInt(30))

	subset = append(subset, big.NewInt(30))
	subset = append(subset, big.NewInt(20))

	sort.Sort(set)
	sort.Sort(subset)

	repr := set.Bits(subset)
	expected := uint64(12) // 0011

	assert.Equal(t, expected, repr)
}

func TestBitIntersect(t *testing.T) {
	set := New()
	subset := New()

	for i := 0; i < 50; i++ {
		k, _ := crypto.RandEntropy(32)
		bk := (&big.Int{}).SetBytes(k)
		set = append(set, bk)
		if rand.Intn(100) < 30 {
			subset = append(subset, bk)
		}
	}

	sort.Sort(set)
	sort.Sort(subset)

	bRepr := set.Bits(subset)
	sub := set.Intersect(bRepr)

	assert.Equal(t, subset, sub)
}

func TestRemove(t *testing.T) {
	nr := 5
	set := New()
	for i := 0; i < nr; i++ {
		k, _ := crypto.RandEntropy(32)
		bk := (&big.Int{}).SetBytes(k)
		set = append(set, bk)
	}
	sort.Sort(set)

	lastElem := set[nr-1].Bytes()
	set.Remove(lastElem)
	i, found := set.IndexOf(lastElem)
	assert.False(t, found)
	assert.Equal(t, nr-1, i)
}

func TestInsert(t *testing.T) {
	v := New()

	v.Insert(big.NewInt(45).Bytes())
	v.Insert(big.NewInt(34).Bytes())
	v.Insert(big.NewInt(63).Bytes())

	assert.Equal(t, 0, big.NewInt(34).Cmp(v[0]))
	assert.Equal(t, 0, big.NewInt(45).Cmp(v[1]))
	assert.Equal(t, 0, big.NewInt(63).Cmp(v[2]))

	assert.Equal(t, 3, len(v))
}

package sortedset

import (
	"math/big"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

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

	assert.True(t, v.Insert(big.NewInt(45).Bytes()))
	assert.True(t, v.Insert(big.NewInt(34).Bytes()))
	assert.True(t, v.Insert(big.NewInt(63).Bytes()))
	assert.False(t, v.Insert(big.NewInt(34).Bytes()))

	assert.Equal(t, 0, big.NewInt(34).Cmp(v[0]))
	assert.Equal(t, 0, big.NewInt(45).Cmp(v[1]))
	assert.Equal(t, 0, big.NewInt(63).Cmp(v[2]))

	assert.Equal(t, 3, len(v))
}

func TestSize(t *testing.T) {
	v := New()

	assert.True(t, v.Insert(big.NewInt(45).Bytes()))
	assert.True(t, v.Insert(big.NewInt(34).Bytes()))
	assert.True(t, v.Insert(big.NewInt(63).Bytes()))
	assert.False(t, v.Insert(big.NewInt(34).Bytes()))

	assert.Equal(t, 3, len(v))
}

package sortedset

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOccurrence(t *testing.T) {
	v := NewCluster()

	v.Insert(big.NewInt(45).Bytes())
	v.Insert(big.NewInt(34).Bytes())
	v.Insert(big.NewInt(34).Bytes())
	v.Insert(big.NewInt(34).Bytes())
	v.Insert(big.NewInt(63).Bytes())

	assert.Equal(t, 0, big.NewInt(34).Cmp(v.Set[0]))
	assert.Equal(t, 0, big.NewInt(45).Cmp(v.Set[1]))
	assert.Equal(t, 0, big.NewInt(63).Cmp(v.Set[2]))

	assert.Equal(t, 3, len(v.Set))

	assert.Equal(t, 3, v.OccurrencesOf(big.NewInt(34).Bytes()))
	assert.Equal(t, 5, v.TotalOccurrences())
}

func TestRemoveFromCluster(t *testing.T) {
	v := NewCluster()

	v.Insert(big.NewInt(45).Bytes())
	v.Insert(big.NewInt(34).Bytes())
	v.Insert(big.NewInt(34).Bytes())
	v.Insert(big.NewInt(34).Bytes())
	v.Insert(big.NewInt(63).Bytes())

	assert.True(t, v.Remove(big.NewInt(34).Bytes()))

	assert.Equal(t, 2, v.OccurrencesOf(big.NewInt(34).Bytes()))
	assert.Equal(t, 4, v.TotalOccurrences())

	assert.Equal(t, 2, v.RemoveAll(big.NewInt(34).Bytes()))
}

func BenchmarkClusterInsert(b *testing.B) {
	v := NewCluster()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		bytes := big.NewInt(int64(rand.Uint64())).Bytes()
		b.StartTimer()
		v.Insert(bytes)
	}
}

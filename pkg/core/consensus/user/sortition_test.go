package user

import (
	"math/big"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBits(t *testing.T) {
	committee := newCommittee()
	subcommittee := newCommittee()

	committee = append(committee, big.NewInt(0))
	committee = append(committee, big.NewInt(10))
	committee = append(committee, big.NewInt(20))
	committee = append(committee, big.NewInt(30))

	subcommittee = append(subcommittee, big.NewInt(30))
	subcommittee = append(subcommittee, big.NewInt(20))

	repr := committee.Bits(subcommittee)
	expected := uint64(12) // 0011

	assert.Equal(t, expected, repr)
}

func TestBitIntersect(t *testing.T) {
	committee := newCommittee()
	subcommittee := newCommittee()
	for i := 0; i < 50; i++ {
		k, _ := NewRandKeys()
		bk := (&big.Int{}).SetBytes(k.BLSPubKey.Marshal())
		committee = append(committee, bk)
		if rand.Intn(100) < 30 {
			subcommittee = append(committee, bk)
		}
	}

	sort.Sort(committee)
	sort.Sort(subcommittee)

	bRepr := committee.Bits(subcommittee)
	sub := committee.Intersect(bRepr)

	assert.Equal(t, subcommittee, sub)
}

func TestRemove(t *testing.T) {
	nr := 5
	committee := newCommittee()
	for i := 0; i < nr; i++ {
		k, _ := NewRandKeys()
		bk := (&big.Int{}).SetBytes(k.BLSPubKey.Marshal())
		committee = append(committee, bk)
	}
	sort.Sort(committee)

	lastElem := committee[nr-1].Bytes()
	committee.Remove(lastElem)
	i, found := committee.IndexOf(lastElem)
	assert.False(t, found)
	assert.Equal(t, nr-1, i)
}

func TestInsert(t *testing.T) {
	v := newCommittee()

	assert.True(t, v.Insert(big.NewInt(45)))
	assert.True(t, v.Insert(big.NewInt(34)))
	assert.True(t, v.Insert(big.NewInt(63)))
	assert.False(t, v.Insert(big.NewInt(34)))

	assert.Equal(t, 0, big.NewInt(34).Cmp(v[0]))
	assert.Equal(t, 0, big.NewInt(45).Cmp(v[1]))
	assert.Equal(t, 0, big.NewInt(63).Cmp(v[2]))

	assert.Equal(t, 3, len(v))
}

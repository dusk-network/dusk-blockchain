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

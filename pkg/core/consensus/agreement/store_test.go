package agreement

import (
	"fmt"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

var hdr header.Header
var blsPubKey, _ = crypto.RandEntropy(32)

func mockAgreement(id string, blockHash []byte, step uint8) Agreement {
	h := hdr
	h.BlockHash = blockHash
	a := Agreement{
		Header: header.Header{
			Round:     uint64(1),
			Step:      step,
			PubKeyBLS: blsPubKey,
		},
	}
	a.SetSignature([]byte(id))
	return a
}

var test = []struct {
	sig              string
	hash             []byte
	step             uint8
	storedAgreements int
}{
	{"pippo", []byte("hash1"), 1, 1},
	{"pluto", []byte("hash2"), 2, 1},
	{"pippo", []byte("hash1"), 1, 1},
	{"paperino", []byte("hash2"), 2, 2},
	{"pippo", []byte("hash2"), 2, 3},
}

func TestStore(t *testing.T) {
	s := newStore()
	steps := make([]uint8, 0)
	for i, tt := range test {
		steps = append(steps, tt.step)
		mock := mockAgreement(tt.sig, tt.hash, tt.step)
		if !assert.Equal(t, tt.storedAgreements, s.Insert(mock, 1)) {
			assert.FailNow(t, fmt.Sprintf("store.Insertion failed at row: %d", i))
		}
		if !assert.True(t, s.Contains(mock)) {
			assert.FailNow(t, fmt.Sprintf("store.Contains failed at row: %d", i))
		}
		if !assert.Equal(t, tt.storedAgreements, len(s.Get(tt.step))) {
			assert.FailNow(t, fmt.Sprintf("store.Get failed at row: %d", i))
		}
	}

	s.Clear()
	for _, step := range steps {
		if !assert.Nil(t, s.Get(step)) {
			assert.FailNow(t, fmt.Sprintf("store.Clear failed to clean 0x%x", step))
		}
	}
}

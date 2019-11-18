package agreement

import (
	"fmt"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

var hdr header.Header

func init() {

	blsPubKey, _ := crypto.RandEntropy(32)
	hdr = header.Header{
		Round:     uint64(1),
		Step:      uint8(1),
		PubKeyBLS: blsPubKey,
	}
}

func mockAgreement(id string, blockHash []byte) Agreement {
	h := hdr
	h.BlockHash = blockHash
	a := Agreement{
		Header: h,
	}
	a.SetSignature([]byte(id))
	return a
}

var test = []struct {
	sig              string
	hash             []byte
	storedAgreements int
}{
	{"pippo", []byte("hash1"), 1},
	{"pluto", []byte("hash2"), 1},
	{"pippo", []byte("hash1"), 1},
	{"paperino", []byte("hash2"), 2},
	{"pippo", []byte("hash2"), 3},
}

func TestStore(t *testing.T) {
	s := newStore()
	hashes := make([][]byte, 0)
	for i, tt := range test {
		hashes = append(hashes, tt.hash)
		mock := mockAgreement(tt.sig, tt.hash)
		if !assert.Equal(t, tt.storedAgreements, s.Insert(mock, 1)) {
			assert.FailNow(t, fmt.Sprintf("store.Insertion failed at row: %d", i))
		}
		if !assert.True(t, s.Contains(mock)) {
			assert.FailNow(t, fmt.Sprintf("store.Contains failed at row: %d", i))
		}
		if !assert.Equal(t, tt.storedAgreements, len(s.Get(tt.hash))) {
			assert.FailNow(t, fmt.Sprintf("store.Get failed at row: %d", i))
		}
	}

	s.Clear()
	for _, hh := range hashes {
		if !assert.Nil(t, s.Get(hh)) {
			assert.FailNow(t, fmt.Sprintf("store.Clear failed to clean 0x%x", hh))
		}
	}
}

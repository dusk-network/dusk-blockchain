package key_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toghrulmaharramov/dusk-go/crypto"
	"github.com/toghrulmaharramov/dusk-go/wallet/key"
)

func TestNewAddress(t *testing.T) {

	en := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	k, err := key.New(en)
	assert.Equal(t, nil, err)

	want := "3PDB7r78p1vNE2hMr5Q4HoEeHE1jthSCbV4nmztXsgiwqcFQhx66mpv7SVZPVrmskq6GrMNWS9kHEpZo7WM6B9b6oLuDiGh"
	got, err := k.Address()
	assert.Equal(t, nil, err)
	assert.Equal(t, want, got)
}

func TestRandomAddresses(t *testing.T) {

	addrLen := 95

	// Test 100 random addresses
	for i := 0; i < 100; i++ {

		// Generate entropy
		en, err := crypto.RandEntropy(32)
		assert.Equal(t, nil, err)

		// Generate keys
		k, err := key.New(en)
		assert.Equal(t, nil, err)

		assert.Equal(t, nil, err)
		assert.Equal(t, key.KeySize, len(k.PrivateSpend))
		assert.Equal(t, key.KeySize, len(k.PublicSpend))
		assert.Equal(t, key.KeySize, len(k.PrivateView))
		assert.Equal(t, key.KeySize, len(k.PublicView))

		a, err := k.Address()
		assert.Equal(t, nil, err)
		assert.Equal(t, len(a), addrLen)

	}
}
func TestBadEntropyLength(t *testing.T) {

	for i := 0; i < 1e5; i++ {

		n := rand.Intn(1e3)
		if n == 32 {
			continue
		}

		en := make([]byte, n)

		k, err := key.New(en)
		assert.NotEqual(t, nil, err)

		if k != nil {
			assert.Fail(t, "key should not be generated when entropy does not equal 32 bytes")
		}

	}

}

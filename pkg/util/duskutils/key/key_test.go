package key_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/key"
)

func TestNewAddress(t *testing.T) {

	en := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	k, err := key.New(en)
	assert.Equal(t, nil, err)

	want := "3PXJRmur5X1FhAD8qRi1UZ5S2np4FUdRikq3Q395LgVQV2cKrM1fC8TG1tDk88ueLrQ5Y9VeGKikJo2GLvNVNTaWk8E8RiB"
	got, err := k.PublicAddress()
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
		assert.Equal(t, key.KeySize, len(k.PublicSpend.Bytes()))
		assert.Equal(t, key.KeySize, len(k.PrivateView))
		assert.Equal(t, key.KeySize, len(k.PublicView.Bytes()))

		a, err := k.PublicAddress()
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

func TestPubAddrToKey(t *testing.T) {

	en, _ := crypto.RandEntropy(32)
	k, _ := key.New(en)
	addr, _ := k.PublicAddress()

	decK, err := key.PubAddrToKey(addr)
	assert.Equal(t, nil, err)

	assert.Equal(t, k.PublicView.Bytes(), decK.PublicView.Bytes())
	assert.Equal(t, k.PublicSpend.Bytes(), decK.PublicSpend.Bytes())
}

func TestStealth(t *testing.T) {
	en, _ := crypto.RandEntropy(32)
	Alice, _ := key.New(en)

	// Generate random Stealth Address
	P, R, err := Alice.StealthAddress()
	assert.Equal(t, nil, err)

	_, ok := Alice.DidReceiveTx(P, R)
	assert.Equal(t, true, ok)
}

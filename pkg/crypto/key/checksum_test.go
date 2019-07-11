package key

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCorrectChecksum(t *testing.T) {
	for i := 0; i < 100; i++ {
		randData := make([]byte, i)
		_, err := rand.Read(randData)
		assert.Nil(t, err)

		cs, err := checksum(randData)
		assert.Nil(t, err)

		assert.True(t, compareChecksum(randData, cs))
	}
}

func TestWrongChecksum(t *testing.T) {
	for i := 0; i < 100; i++ {
		randData := make([]byte, i)
		_, err := rand.Read(randData)
		assert.Nil(t, err)

		wrongChecksum := rand.Uint32()

		assert.False(t, compareChecksum(randData, wrongChecksum))
	}
}

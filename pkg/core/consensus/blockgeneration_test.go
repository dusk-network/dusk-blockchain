package consensus

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestGenerateM(t *testing.T) {
	k := []byte{0, 1, 1}
	expect := "c66263a231f51157ce892c54031eed87822c534e9e6dd5471404ef2cabe4f2f7"

	hash, err := generateM(k)
	assert.Equal(t, nil, err)

	assert.Equal(t, expect, hex.EncodeToString(hash))
}
func TestGenerateY(t *testing.T) {

	d := uint64(20)
	S := []byte{0, 1, 1, 1, 1, 1}
	k := []byte{0, 1, 1}
	expect := "bfe1730438950d94a35925c3ca6f1a92465757ff4fce78a5bf34ce6345a6c4ab"

	hash, err := generateY(d, S, k)
	assert.Equal(t, nil, err)

	assert.Equal(t, expect, hex.EncodeToString(hash))
}

func TestGenerateX(t *testing.T) {
	d := uint64(20)
	k := []byte{0, 1, 1}
	expected := "fbad032f388c8465aa271f787a00f4fcf75830b4877c15c1ac4c096eb60cd50d"

	hash, err := generateX(d, k)

	assert.Equal(t, nil, err)
	assert.Equal(t, expected, hex.EncodeToString(hash))
}

//XXX: Add fixed test input vectors to outputs
func TestBlockGeneration(t *testing.T) {

	for i := 0; i < 1000; i++ {

		ctx, err := NewContext(200, randtestKeys(t))
		assert.Equal(t, nil, err)

		k, err := crypto.RandEntropy(32)
		assert.Equal(t, err, nil)

		msgCan, candidateBlock, err := GenerateBlock(ctx, k)
		assert.Equal(t, nil, err)
		assert.NotEqual(t, nil, msgCan)
		assert.NotEqual(t, nil, candidateBlock)

	}
}

// helper to generate random consensus keys
func randtestKeys(t *testing.T) *Keys {
	keys, err := NewRandKeys()
	if err != nil {
		t.FailNow()
	}
	return keys
}

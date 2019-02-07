package consensus_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
)

func TestNewKeys(t *testing.T) {

	for i := 0; i < 100; i++ {
		keys, err := consensus.NewRandKeys()
		assert.Equal(t, nil, err)
		assert.NotEqual(t, nil, keys)
		assert.NotEqual(t, nil, keys.BLSPubKey)
		assert.NotEqual(t, nil, keys.BLSSecretKey)
		assert.NotEqual(t, nil, keys.EdPubKey)
		assert.NotEqual(t, nil, keys.EdSecretKey)
	}

}
func TestClear(t *testing.T) {

	keys, err := consensus.NewRandKeys()
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, keys)
	assert.NotEqual(t, nil, keys.BLSPubKey)
	assert.NotEqual(t, nil, keys.BLSSecretKey)
	assert.NotEqual(t, nil, keys.EdPubKey)
	assert.NotEqual(t, nil, keys.EdSecretKey)

	keys.Clear()
	assert.Nil(t, keys.BLSPubKey, nil, nil)
	assert.Nil(t, keys.BLSSecretKey, nil, nil)
	assert.Nil(t, keys.EdPubKey, nil, nil)
	assert.Nil(t, keys.EdSecretKey, nil, nil)

}

package transactions

import (
	"testing"

	"github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/mlsag"
)

func TestNewInput(t *testing.T) {
	var amount, mask, privkey, commToZero ristretto.Scalar
	amount.Rand()
	mask.Rand()
	privkey.Rand()
	commToZero.Rand()

	i := NewInput(amount, mask, privkey)

	for k := 1; k <= 10; k++ {
		i.Proof.AddDecoy(generateDualKey())
		assert.Equal(t, i.Proof.LenMembers(), k)
	}

	i.Proof.SetCommToZero(commToZero)

	err := i.Prove()
	assert.NotNil(t, i.Signature)
	assert.NotNil(t, i.KeyImage)
	assert.Nil(t, err)
}

func generateDualKey() mlsag.PubKeys {
	pubkeys := mlsag.PubKeys{}

	var primaryKey ristretto.Point
	primaryKey.Rand()
	pubkeys.AddPubKey(primaryKey)

	var secondaryKey ristretto.Point
	secondaryKey.Rand()
	pubkeys.AddPubKey(secondaryKey)

	return pubkeys
}

func generateDecoys(numMixins int) []mlsag.PubKeys {

	var pubKeys []mlsag.PubKeys
	for i := 0; i < numMixins; i++ {
		pubKeyVector := generateDualKey()
		pubKeys = append(pubKeys, pubKeyVector)
	}
	return pubKeys
}

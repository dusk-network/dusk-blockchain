package agreement

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
)

func TestStepVotes(t *testing.T) {
	set := sortedset.New()

	hash := []byte("this is a mock message")
	k1 := genKeys(&set)
	k2 := genKeys(&set)
	subset := set
	//inserting a third key in the set to better test packing and unpacking
	genKeys(&set)

	bitset := set.Bits(subset)

	s, err := bls.Sign(k1.BLSSecretKey, k1.BLSPubKey, hash)
	assert.NoError(t, err)

	s2, err := bls.Sign(k2.BLSSecretKey, k2.BLSPubKey, hash)
	assert.NoError(t, err)

	apk := bls.NewApk(k1.BLSPubKey)
	assert.NoError(t, apk.Aggregate(k2.BLSPubKey))

	s.Aggregate(s2)
	assert.NoError(t, bls.Verify(apk, hash, s))

	expectedStepVotes := NewStepVotes()
	expectedStepVotes.Apk = apk
	expectedStepVotes.BitSet = bitset
	expectedStepVotes.Signature = s

	buf := new(bytes.Buffer)

	assert.NoError(t, MarshalStepVotes(buf, expectedStepVotes))

	result, err := UnmarshalStepVotes(buf)
	assert.NoError(t, err)

	assert.Equal(t, expectedStepVotes, result)
	assert.NoError(t, bls.Verify(result.Apk, hash, result.Signature))
}

// Test that adding Reduction events to a StepVotes struct results in a properly
// aggregated public key and signature.
func TestStepVotesAdd(t *testing.T) {
	sv := NewStepVotes()
	set := sortedset.New()
	hash := []byte("this is a mock message")
	assert.NoError(t, sv.Add(genReduction(hash, &set)))
	assert.NoError(t, sv.Add(genReduction(hash, &set)))
	assert.NoError(t, sv.Add(genReduction(hash, &set)))

	assert.NoError(t, bls.Verify(sv.Apk, hash, sv.Signature))
}

func genKeys(set *sortedset.Set) user.Keys {
	k, _ := user.NewRandKeys()
	set.Insert(k.BLSPubKeyBytes)
	return k
}

func genReduction(hash []byte, set *sortedset.Set) ([]byte, []byte, uint8) {
	k := genKeys(set)
	s, err := bls.Sign(k.BLSSecretKey, k.BLSPubKey, hash)
	if err != nil {
		panic(err)
	}

	return s.Compress(), k.BLSPubKeyBytes, uint8(1)
}

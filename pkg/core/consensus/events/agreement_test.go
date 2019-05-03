package events

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
)

func TestStepVotes(t *testing.T) {
	set := sortedset.New()

	hash := []byte("this is a mock message")
	pk1, sk1 := genKeys(&set)
	pk2, sk2 := genKeys(&set)
	subset := set
	//inserting a third key in the set to better test packing and unpacking
	genKeys(&set)

	bitset := set.Bits(subset)

	s, err := bls.Sign(sk1, pk1, hash)
	assert.NoError(t, err)

	s2, err := bls.Sign(sk2, pk2, hash)
	assert.NoError(t, err)

	apk := bls.NewApk(pk1)
	assert.NoError(t, apk.Aggregate(pk2))

	s.Aggregate(s2)
	assert.NoError(t, bls.Verify(apk, hash, s))

	expectedStepVotes := NewStepVotes(apk.Marshal(), bitset, s.Compress())
	buf := new(bytes.Buffer)

	assert.NoError(t, MarshalStepVotes(buf, expectedStepVotes))

	result, err := UnmarshalStepVotes(buf)
	assert.NoError(t, err)

	assert.Equal(t, expectedStepVotes, result)

	apkRes, err := bls.UnmarshalApk(result.Apk)
	assert.NoError(t, err)
	sigma, err := bls.UnmarshalSignature(result.Signature)
	assert.NoError(t, err)

	assert.NoError(t, bls.Verify(apkRes, hash, sigma))
}

func genKeys(set *sortedset.Set) (*bls.PublicKey, *bls.SecretKey) {
	pk, sk, _ := bls.GenKeyPair(rand.Reader)
	set.Insert(pk.Marshal())
	return pk, sk
}

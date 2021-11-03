// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message
import (
	"bytes"
	"testing"

	"github.com/dusk-network/bls12_381-sign/bls12_381-sign-go/bls"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/stretchr/testify/assert"
)

func TestStepVotes(t *testing.T) {
	set := sortedset.New()

	hash := []byte("this is a mock message")
	k1 := genKeys(&set)
	k2 := genKeys(&set)
	subset := set
	// inserting a third key in the set to better test packing and unpacking
	genKeys(&set)

	bitset := set.Bits(subset)

	s, err := bls.Sign(k1.BLSSecretKey, k1.BLSPubKey, hash)
	assert.NoError(t, err)

	s2, err := bls.Sign(k2.BLSSecretKey, k2.BLSPubKey, hash)
	assert.NoError(t, err)

	apk, err := bls.CreateApk(k1.BLSPubKey)
	assert.NoError(t, err)

	apk, err = bls.AggregatePk(apk, k2.BLSPubKey)
	assert.NoError(t, err)

	s3, err := bls.AggregateSig(s, s2)
	assert.NoError(t, err)

	assert.NoError(t, bls.Verify(apk, s3, hash))

	expectedStepVotes := NewStepVotes()
	expectedStepVotes.BitSet = bitset
	expectedStepVotes.Signature = s3

	buf := new(bytes.Buffer)

	assert.NoError(t, MarshalStepVotes(buf, expectedStepVotes))

	result, err := UnmarshalStepVotes(buf)
	assert.NoError(t, err)

	assert.Equal(t, expectedStepVotes, result)
	assert.NoError(t, bls.Verify(apk, result.Signature, hash))
}

// Test that adding Reduction events to a StepVotes struct results in a properly
// aggregated public key and signature.
func TestStepVotesAdd(t *testing.T) {
	sv := NewStepVotes()
	set := sortedset.New()
	hash := []byte("this is a mock message")
	signedHash, pk := genReduction(hash, &set)

	apk, err := bls.CreateApk(pk)
	assert.NoError(t, err)

	assert.NoError(t, sv.Add(signedHash))

	signedHash, pk = genReduction(hash, &set)

	apk, err = bls.AggregatePk(apk, pk)
	assert.NoError(t, err)

	assert.NoError(t, sv.Add(signedHash))

	signedHash, pk = genReduction(hash, &set)

	apk, err = bls.AggregatePk(apk, pk)
	assert.NoError(t, err)

	assert.NoError(t, sv.Add(signedHash))

	assert.NoError(t, bls.Verify(apk, sv.Signature, hash))
}

func genKeys(set *sortedset.Set) key.Keys {
	k := key.NewRandKeys()
	set.Insert(k.BLSPubKey)
	return k
}

func genReduction(hash []byte, set *sortedset.Set) ([]byte, []byte) {
	k := genKeys(set)

	s, err := bls.Sign(k.BLSSecretKey, k.BLSPubKey, hash)
	if err != nil {
		panic(err)
	}

	return s, k.BLSPubKey
}
